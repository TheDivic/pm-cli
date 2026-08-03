// Package query flattens discovered tasks and filters them. Filters combine
// with logical AND across categories; repeated values within a category combine
// with logical OR. File order is preserved so the first ready task in the
// result is the next one to work on.
package query

import (
	"sort"

	"github.com/TheDivic/plaintext-projects/internal/discover"
	"github.com/TheDivic/plaintext-projects/internal/model"
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

// statusRank orders task statuses for list display: active work first, with
// in-review ahead of in-progress since it is closer to completion, then
// upcoming (todo, backlog), then closed (done, cancelled).
func statusRank(s model.TaskStatus) int {
	switch s {
	case model.TaskInReview:
		return 0
	case model.TaskInProgress:
		return 1
	case model.TaskTodo:
		return 2
	case model.TaskBacklog:
		return 3
	case model.TaskDone:
		return 4
	case model.TaskCancelled:
		return 5
	default:
		return 6
	}
}

// SortForList stably orders refs by status (active work first), then by
// explicit task priority (lowest number first, tasks without a priority last).
// Ties keep the incoming file order, which acts as the within-status priority.
func SortForList(refs []TaskRef) {
	sort.SliceStable(refs, func(i, j int) bool {
		si, sj := statusRank(refs[i].Task.Status), statusRank(refs[j].Task.Status)
		if si != sj {
			return si < sj
		}
		return priorityLess(refs[i].Task.Priority, refs[j].Task.Priority)
	})
}

// priorityLess orders by priority with lower numbers first and unset last.
// Equal or both-unset priorities compare equal so a stable sort keeps file
// order.
func priorityLess(a, b *int) bool {
	if (a == nil) != (b == nil) {
		return a != nil // a set, b unset -> a first
	}
	if a != nil && b != nil && *a != *b {
		return *a < *b
	}
	return false
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
