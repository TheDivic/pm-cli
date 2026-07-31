// Package validate applies schema version 1 semantic rules to a decoded
// document and returns every independent finding in one pass.
//
// Scope note: this pass covers schema version, required fields, value formats,
// enum values, task-ID format and uniqueness, terminal-status consistency,
// blocking rules, and parent existence and cycles. Blocker-cycle detection,
// area-file existence, and canonical field-order checks are handled elsewhere
// or in a later pass.
package validate

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/TheDivic/plaintext-tasks/internal/model"
)

// Finding is one validation problem with a dotted field path and, when the
// problem concerns a task, its ID.
type Finding struct {
	Field   string
	Task    string
	Message string
}

func (f Finding) String() string {
	loc := f.Field
	if f.Task != "" {
		loc = f.Task + " " + loc
	}
	if loc == "" {
		return f.Message
	}
	return f.Message + " (" + loc + ")"
}

var (
	kebabRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	dateRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Document validates a decoded document and returns all findings. An empty
// slice means the document is valid.
func Document(doc *model.Document) []Finding {
	var f []Finding
	if doc.SchemaVersion != model.SchemaVersion {
		f = append(f, Finding{Field: "schema-version", Message: fmt.Sprintf("unsupported schema version %d (only %d is supported)", doc.SchemaVersion, model.SchemaVersion)})
	}
	f = append(f, validateProject(&doc.Project)...)
	f = append(f, validateTasks(doc)...)
	return f
}

func validateProject(p *model.Project) []Finding {
	var f []Finding
	add := func(field, msg string) { f = append(f, Finding{Field: "project." + field, Message: msg}) }

	if p.ID == "" {
		add("id", "is required")
	} else if !kebabRe.MatchString(p.ID) {
		add("id", "must be lowercase kebab-case")
	}
	if strings.TrimSpace(p.Title) == "" {
		add("title", "is required")
	} else if strings.ContainsAny(p.Title, "\n") {
		add("title", "must be a single line")
	}
	if p.TaskIDPrefix == "" {
		add("task-id-prefix", "is required")
	} else if !kebabRe.MatchString(p.TaskIDPrefix) {
		add("task-id-prefix", "must be lowercase kebab-case")
	}
	if p.Status == "" {
		add("status", "is required")
	} else if !p.Status.Valid() {
		add("status", fmt.Sprintf("%q is not a valid project status", p.Status))
	}
	if p.Priority != nil && *p.Priority < 1 {
		add("priority", "must be a positive integer")
	}
	f = append(f, validateAreas(p.Areas, "project.areas")...)

	if p.Created == "" {
		add("created", "is required")
	} else if !validDate(p.Created) {
		add("created", "must be a valid YYYY-MM-DD date")
	}
	for field, val := range map[string]string{"started": p.Started, "due": p.Due, "completed": p.Completed} {
		if val != "" && !validDate(val) {
			add(field, "must be a valid YYYY-MM-DD date")
		}
	}
	f = append(f, validateBlocked(p.Blocked, "project.blocked")...)
	f = append(f, validateCancellation(p.Cancellation, "project.cancellation")...)

	// Terminal-status consistency for the project.
	switch p.Status {
	case model.ProjectDone:
		if p.Completed == "" {
			add("completed", "is required when status is done")
		}
	case model.ProjectCancelled:
		if p.Cancellation == nil {
			add("cancellation", "is required when status is cancelled")
		}
	case model.ProjectBlocked:
		if p.Blocked == nil {
			add("blocked", "is required when status is blocked")
		}
	}
	return f
}

func validateTasks(doc *model.Document) []Finding {
	var f []Finding
	prefix := doc.Project.TaskIDPrefix
	idRe := taskIDPattern(prefix)

	ids := make(map[string]bool, len(doc.Tasks))
	for i := range doc.Tasks {
		t := &doc.Tasks[i]
		path := fmt.Sprintf("tasks[%d]", i)
		add := func(field, msg string) {
			f = append(f, Finding{Field: path + "." + field, Task: t.ID, Message: msg})
		}

		switch {
		case t.ID == "":
			add("id", "is required")
		case idRe != nil && !idRe.MatchString(t.ID):
			add("id", fmt.Sprintf("must match %q followed by at least three digits", prefix+"-"))
		case ids[t.ID]:
			add("id", "duplicates another task ID")
		default:
			ids[t.ID] = true
		}

		if strings.TrimSpace(t.Title) == "" {
			add("title", "is required")
		} else if strings.ContainsAny(t.Title, "\n") {
			add("title", "must be a single line")
		}
		if t.Status == "" {
			add("status", "is required")
		} else if !t.Status.Valid() {
			add("status", fmt.Sprintf("%q is not a valid task status", t.Status))
		}
		if t.Priority != nil && *t.Priority < 1 {
			add("priority", "must be a positive integer")
		}
		if t.Created == "" {
			add("created", "is required")
		} else if !validDate(t.Created) {
			add("created", "must be a valid YYYY-MM-DD date")
		}
		for field, val := range map[string]string{"started": t.Started, "due": t.Due, "completed": t.Completed} {
			if val != "" && !validDate(val) {
				add(field, "must be a valid YYYY-MM-DD date")
			}
		}
		f = append(f, validateAreas(t.Tags, path+".tags")...) // tags share the kebab+unique rule
		f = append(f, tagField(validateBlocked(t.Blocked, path+".blocked"), t.ID)...)
		f = append(f, tagField(validateCancellation(t.Cancellation, path+".cancellation"), t.ID)...)

		// Terminal-status consistency.
		switch {
		case t.Status == model.TaskDone && t.Completed == "":
			add("completed", "is required when status is done")
		case t.Status == model.TaskCancelled && t.Cancellation == nil:
			add("cancellation", "is required when status is cancelled")
		case !t.Status.Terminal() && t.Completed != "":
			add("completed", "is only allowed when status is done")
		case !t.Status.Terminal() && t.Cancellation != nil:
			add("cancellation", "is only allowed when status is cancelled")
		}

		// Blocking is invalid on backlog, cancelled, and done tasks.
		if t.Blocked != nil {
			switch t.Status {
			case model.TaskBacklog, model.TaskCancelled, model.TaskDone:
				add("blocked", fmt.Sprintf("is not allowed when status is %s", t.Status))
			}
		}
	}

	f = append(f, validateReferences(doc, ids)...)
	return f
}

// validateReferences checks parent and blocker references against the set of
// existing task IDs and detects parent cycles.
func validateReferences(doc *model.Document, ids map[string]bool) []Finding {
	var f []Finding
	parent := make(map[string]string, len(doc.Tasks))
	for i := range doc.Tasks {
		t := &doc.Tasks[i]
		path := fmt.Sprintf("tasks[%d]", i)
		if t.ID != "" {
			parent[t.ID] = t.Parent
		}
		if t.Parent != "" {
			switch {
			case t.Parent == t.ID:
				f = append(f, Finding{Field: path + ".parent", Task: t.ID, Message: "a task cannot be its own parent"})
			case !ids[t.Parent]:
				f = append(f, Finding{Field: path + ".parent", Task: t.ID, Message: fmt.Sprintf("references unknown task %q", t.Parent)})
			}
		}
		if t.Blocked != nil {
			seen := map[string]bool{}
			for j, b := range t.Blocked.Tasks {
				bf := fmt.Sprintf("%s.blocked.tasks[%d]", path, j)
				switch {
				case b == t.ID:
					f = append(f, Finding{Field: bf, Task: t.ID, Message: "a task cannot block itself"})
				case !ids[b]:
					f = append(f, Finding{Field: bf, Task: t.ID, Message: fmt.Sprintf("references unknown task %q", b)})
				case seen[b]:
					f = append(f, Finding{Field: bf, Task: t.ID, Message: fmt.Sprintf("duplicates blocker %q", b)})
				}
				seen[b] = true
			}
		}
	}
	f = append(f, parentCycles(doc, parent)...)
	return f
}

// parentCycles reports each task that sits on a parent-reference cycle.
func parentCycles(doc *model.Document, parent map[string]string) []Finding {
	var f []Finding
	for i := range doc.Tasks {
		id := doc.Tasks[i].ID
		if id == "" {
			continue
		}
		visited := map[string]bool{id: true}
		cur := parent[id]
		for cur != "" {
			if cur == id {
				f = append(f, Finding{Field: fmt.Sprintf("tasks[%d].parent", i), Task: id, Message: "is part of a parent cycle"})
				break
			}
			if visited[cur] {
				break // cycle not involving id; the involved task reports it itself
			}
			visited[cur] = true
			cur = parent[cur]
		}
	}
	return f
}

func validateAreas(values []string, path string) []Finding {
	var f []Finding
	seen := map[string]bool{}
	for i, v := range values {
		p := fmt.Sprintf("%s[%d]", path, i)
		if !kebabRe.MatchString(v) {
			f = append(f, Finding{Field: p, Message: "must be lowercase kebab-case"})
		}
		if seen[v] {
			f = append(f, Finding{Field: p, Message: fmt.Sprintf("duplicate value %q", v)})
		}
		seen[v] = true
	}
	return f
}

func validateBlocked(b *model.Blocked, path string) []Finding {
	if b == nil {
		return nil
	}
	var f []Finding
	if strings.TrimSpace(b.Reason) == "" {
		f = append(f, Finding{Field: path + ".reason", Message: "is required"})
	}
	if b.Since == "" {
		f = append(f, Finding{Field: path + ".since", Message: "is required"})
	} else if !validDate(b.Since) {
		f = append(f, Finding{Field: path + ".since", Message: "must be a valid YYYY-MM-DD date"})
	}
	return f
}

func validateCancellation(c *model.Cancellation, path string) []Finding {
	if c == nil {
		return nil
	}
	var f []Finding
	if strings.TrimSpace(c.Reason) == "" {
		f = append(f, Finding{Field: path + ".reason", Message: "is required"})
	}
	if c.Date == "" {
		f = append(f, Finding{Field: path + ".date", Message: "is required"})
	} else if !validDate(c.Date) {
		f = append(f, Finding{Field: path + ".date", Message: "must be a valid YYYY-MM-DD date"})
	}
	return f
}

// tagField stamps a task ID onto findings produced by a shared helper.
func tagField(f []Finding, task string) []Finding {
	for i := range f {
		if f[i].Task == "" {
			f[i].Task = task
		}
	}
	return f
}

func taskIDPattern(prefix string) *regexp.Regexp {
	if prefix == "" {
		return nil
	}
	return regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `-\d{3,}$`)
}

func validDate(s string) bool {
	if !dateRe.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}
