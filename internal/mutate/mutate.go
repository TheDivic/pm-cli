// Package mutate holds the pure in-memory transforms that back the mutation
// commands. Each function applies exactly one requested change to a document;
// the caller validates before and after and writes atomically.
package mutate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TheDivic/plaintext-tasks/internal/clock"
	"github.com/TheDivic/plaintext-tasks/internal/model"
)

// NextTaskID returns the next task ID for the project's prefix. It scans every
// task (including completed and cancelled ones) and pads the sequence to at
// least three digits.
func NextTaskID(doc *model.Document) string {
	prefix := doc.Project.TaskIDPrefix
	highest := 0
	for i := range doc.Tasks {
		id := doc.Tasks[i].ID
		if !strings.HasPrefix(id, prefix+"-") {
			continue
		}
		if n, err := strconv.Atoi(id[len(prefix)+1:]); err == nil && n > highest {
			highest = n
		}
	}
	seq := strconv.Itoa(highest + 1)
	for len(seq) < 3 {
		seq = "0" + seq
	}
	return prefix + "-" + seq
}

func findTask(doc *model.Document, id string) (int, error) {
	for i := range doc.Tasks {
		if doc.Tasks[i].ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no task with id %q in this project", id)
}

// ---- projects ----

// ProjectEdit describes the mutable project fields for a general edit.
type ProjectEdit struct {
	Title         *string
	Priority      *int
	ClearPriority bool
	Due           *string
	ClearDue      bool
	AddAreas      []string
	RemoveAreas   []string
}

// EditProject changes title, priority, due date, and areas. It never changes
// the project ID, task-ID prefix, status, or lifecycle dates.
func EditProject(doc *model.Document, e ProjectEdit) error {
	p := &doc.Project
	if e.Title != nil {
		p.Title = *e.Title
	}
	switch {
	case e.ClearPriority:
		p.Priority = nil
	case e.Priority != nil:
		p.Priority = e.Priority
	}
	switch {
	case e.ClearDue:
		p.Due = ""
	case e.Due != nil:
		p.Due = *e.Due
	}
	for _, a := range e.AddAreas {
		if !contains(p.Areas, a) {
			p.Areas = append(p.Areas, a)
		}
	}
	if len(e.RemoveAreas) > 0 {
		p.Areas = without(p.Areas, e.RemoveAreas)
	}
	return nil
}

// ProjectStatus applies a lifecycle transition and manages its dates.
func ProjectStatus(doc *model.Document, status model.ProjectStatus, reason string, clk clock.Clock) error {
	if !status.Valid() {
		return fmt.Errorf("%q is not a valid project status", status)
	}
	p := &doc.Project
	today := clk.Today()
	switch status {
	case model.ProjectInProgress:
		if p.Started == "" {
			p.Started = today
		}
	case model.ProjectDone:
		p.Completed = today
	case model.ProjectBlocked:
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("entering blocked requires a reason")
		}
		p.Blocked = &model.Blocked{Reason: reason, Since: today}
	case model.ProjectCancelled:
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("entering cancelled requires a reason")
		}
		p.Cancellation = &model.Cancellation{Reason: reason, Date: today}
	}
	if status != model.ProjectBlocked {
		p.Blocked = nil // leaving blocked removes the blocking record
	}
	p.Status = status
	return nil
}

// ---- tasks ----

// TaskAdd describes a new task.
type TaskAdd struct {
	Title       string
	Description string
	Status      model.TaskStatus
	Priority    *int
	Parent      string
	Due         string
	Tags        []string
}

// AddTask allocates the next ID, stamps the created date, and inserts the task.
// A child is placed immediately after its parent's subtree; a root task is
// appended.
func AddTask(doc *model.Document, a TaskAdd, clk clock.Clock) (string, error) {
	status := a.Status
	if status == "" {
		status = model.TaskBacklog
	}
	t := model.Task{
		ID:          NextTaskID(doc),
		Title:       a.Title,
		Description: a.Description,
		Status:      status,
		Priority:    a.Priority,
		Parent:      a.Parent,
		Created:     clk.Today(),
		Due:         a.Due,
		Tags:        a.Tags,
	}
	at := insertIndex(doc, a.Parent)
	doc.Tasks = append(doc.Tasks, model.Task{})
	copy(doc.Tasks[at+1:], doc.Tasks[at:])
	doc.Tasks[at] = t
	return t.ID, nil
}

// insertIndex returns where to place a new task: after the parent's subtree, or
// at the end for a root task or an unknown parent (validation flags the latter).
func insertIndex(doc *model.Document, parent string) int {
	if parent == "" {
		return len(doc.Tasks)
	}
	pi := -1
	for i := range doc.Tasks {
		if doc.Tasks[i].ID == parent {
			pi = i
			break
		}
	}
	if pi < 0 {
		return len(doc.Tasks)
	}
	i := pi + 1
	for i < len(doc.Tasks) && descendantOf(doc, doc.Tasks[i].ID, parent) {
		i++
	}
	return i
}

func descendantOf(doc *model.Document, id, ancestor string) bool {
	parents := map[string]string{}
	for i := range doc.Tasks {
		parents[doc.Tasks[i].ID] = doc.Tasks[i].Parent
	}
	for cur := parents[id]; cur != ""; cur = parents[cur] {
		if cur == ancestor {
			return true
		}
	}
	return false
}

// TaskEdit describes the mutable task fields for a general edit.
type TaskEdit struct {
	Title         *string
	Description   *string
	Priority      *int
	ClearPriority bool
	Due           *string
	ClearDue      bool
	AddTags       []string
	RemoveTags    []string
	Parent        *string
	ClearParent   bool
}

// EditTask changes title, description, priority, due date, tags, and parent. It
// never changes status or lifecycle dates.
func EditTask(doc *model.Document, id string, e TaskEdit) error {
	i, err := findTask(doc, id)
	if err != nil {
		return err
	}
	t := &doc.Tasks[i]
	if e.Title != nil {
		t.Title = *e.Title
	}
	if e.Description != nil {
		t.Description = *e.Description
	}
	switch {
	case e.ClearPriority:
		t.Priority = nil
	case e.Priority != nil:
		t.Priority = e.Priority
	}
	switch {
	case e.ClearDue:
		t.Due = ""
	case e.Due != nil:
		t.Due = *e.Due
	}
	switch {
	case e.ClearParent:
		t.Parent = ""
	case e.Parent != nil:
		t.Parent = *e.Parent
	}
	for _, tag := range e.AddTags {
		if !contains(t.Tags, tag) {
			t.Tags = append(t.Tags, tag)
		}
	}
	if len(e.RemoveTags) > 0 {
		t.Tags = without(t.Tags, e.RemoveTags)
	}
	return nil
}

// TaskStatus applies a lifecycle transition and manages its dates and the
// mutually exclusive terminal and blocking fields.
func TaskStatus(doc *model.Document, id string, status model.TaskStatus, reason string, clk clock.Clock) error {
	i, err := findTask(doc, id)
	if err != nil {
		return err
	}
	if !status.Valid() {
		return fmt.Errorf("%q is not a valid task status", status)
	}
	t := &doc.Tasks[i]
	today := clk.Today()
	switch status {
	case model.TaskInProgress:
		if t.Started == "" {
			t.Started = today
		}
	case model.TaskDone:
		t.Completed = today
	case model.TaskCancelled:
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("entering cancelled requires a reason")
		}
		t.Cancellation = &model.Cancellation{Reason: reason, Date: today}
	}
	if status != model.TaskDone {
		t.Completed = ""
	}
	if status != model.TaskCancelled {
		t.Cancellation = nil
	}
	// Blocking is invalid on backlog, cancelled, and done.
	if status == model.TaskBacklog || status == model.TaskCancelled || status == model.TaskDone {
		t.Blocked = nil
	}
	t.Status = status
	return nil
}

// BlockTask records a blocking condition without changing the task status.
func BlockTask(doc *model.Document, id, reason string, blockers []string, clk clock.Clock) error {
	i, err := findTask(doc, id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("blocking requires a reason")
	}
	t := &doc.Tasks[i]
	switch t.Status {
	case model.TaskBacklog, model.TaskCancelled, model.TaskDone:
		return fmt.Errorf("cannot block a %s task", t.Status)
	}
	t.Blocked = &model.Blocked{Reason: reason, Tasks: blockers, Since: clk.Today()}
	return nil
}

// UnblockTask removes the complete blocking record.
func UnblockTask(doc *model.Document, id string) error {
	i, err := findTask(doc, id)
	if err != nil {
		return err
	}
	if doc.Tasks[i].Blocked == nil {
		return fmt.Errorf("task %q is not blocked", id)
	}
	doc.Tasks[i].Blocked = nil
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

func without(items, remove []string) []string {
	out := items[:0:0]
	for _, it := range items {
		if !contains(remove, it) {
			out = append(out, it)
		}
	}
	return out
}
