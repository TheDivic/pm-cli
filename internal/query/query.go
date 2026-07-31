// Package query flattens discovered tasks and filters them. Filters combine
// with logical AND across categories; repeated values within a category combine
// with logical OR. File order is preserved so the first ready task in the
// result is the next one to work on.
package query

import (
	"github.com/TheDivic/plaintext-tasks/internal/discover"
	"github.com/TheDivic/plaintext-tasks/internal/model"
)

// TaskRef locates a task within its project and file.
type TaskRef struct {
	Task    *model.Task
	Project *model.Project
	Path    string
}

// Flatten returns every task from successfully decoded projects, preserving
// each project's file order and iterating projects in discovery (ID) order.
func Flatten(ws *discover.Workspace) []TaskRef {
	var refs []TaskRef
	for i := range ws.Projects {
		p := &ws.Projects[i]
		if p.Doc == nil {
			continue
		}
		for j := range p.Doc.Tasks {
			refs = append(refs, TaskRef{
				Task:    &p.Doc.Tasks[j],
				Project: &p.Doc.Project,
				Path:    p.Path,
			})
		}
	}
	return refs
}

// TaskFilter is a conjunction of task filters. Slice fields are disjunctions.
type TaskFilter struct {
	Projects  []string
	Statuses  []string
	Tags      []string
	Areas     []string
	Parent    string
	HasParent bool  // whether the Parent filter is active
	Blocked   *bool // nil means "no blocked filter"
	DueBefore string
	DueOn     string
}

// Filter returns the refs that satisfy f, preserving order.
func Filter(refs []TaskRef, f TaskFilter) []TaskRef {
	out := make([]TaskRef, 0, len(refs))
	for _, r := range refs {
		if f.match(r) {
			out = append(out, r)
		}
	}
	return out
}

func (f TaskFilter) match(r TaskRef) bool {
	t := r.Task
	if len(f.Projects) > 0 && !contains(f.Projects, r.Project.ID) {
		return false
	}
	if len(f.Statuses) > 0 && !contains(f.Statuses, string(t.Status)) {
		return false
	}
	if len(f.Tags) > 0 && !overlaps(f.Tags, t.Tags) {
		return false
	}
	if len(f.Areas) > 0 && !overlaps(f.Areas, r.Project.Areas) {
		return false
	}
	if f.HasParent && t.Parent != f.Parent {
		return false
	}
	if f.Blocked != nil && (t.Blocked != nil) != *f.Blocked {
		return false
	}
	if f.DueOn != "" && t.Due != f.DueOn {
		return false
	}
	if f.DueBefore != "" && (t.Due == "" || t.Due >= f.DueBefore) {
		return false
	}
	return true
}

// Find returns the ref for a task by its global ID, or nil if not found.
func Find(refs []TaskRef, id string) *TaskRef {
	for i := range refs {
		if refs[i].Task.ID == id {
			return &refs[i]
		}
	}
	return nil
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

func overlaps(want, have []string) bool {
	for _, w := range want {
		if contains(have, w) {
			return true
		}
	}
	return false
}
