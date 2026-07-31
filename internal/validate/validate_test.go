package validate

import (
	"strings"
	"testing"

	"github.com/TheDivic/plaintext-tasks/internal/model"
)

func validDoc() *model.Document {
	return &model.Document{
		SchemaVersion: 1,
		Project: model.Project{
			ID:           "demo",
			Title:        "Demo",
			TaskIDPrefix: "dm",
			Status:       model.ProjectInProgress,
			Created:      "2026-07-31",
			Started:      "2026-07-31",
		},
		Tasks: []model.Task{
			{ID: "dm-001", Title: "One", Status: model.TaskTodo, Created: "2026-07-31"},
		},
	}
}

func TestValidDocHasNoFindings(t *testing.T) {
	if f := Document(validDoc()); len(f) != 0 {
		t.Fatalf("expected no findings, got: %v", f)
	}
}

func hasField(f []Finding, field string) bool {
	for _, x := range f {
		if x.Field == field {
			return true
		}
	}
	return false
}

func TestFieldLevelFindings(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*model.Document)
		field  string
	}{
		{"bad schema version", func(d *model.Document) { d.SchemaVersion = 2 }, "schema-version"},
		{"missing project id", func(d *model.Document) { d.Project.ID = "" }, "project.id"},
		{"non-kebab prefix", func(d *model.Document) { d.Project.TaskIDPrefix = "DM_x" }, "project.task-id-prefix"},
		{"bad project status", func(d *model.Document) { d.Project.Status = "weird" }, "project.status"},
		{"zero priority", func(d *model.Document) { z := 0; d.Project.Priority = &z }, "project.priority"},
		{"bad created date", func(d *model.Document) { d.Project.Created = "2026-13-40" }, "project.created"},
		{"done needs completed", func(d *model.Document) { d.Project.Status = model.ProjectDone }, "project.completed"},
		{"bad task id format", func(d *model.Document) { d.Tasks[0].ID = "xx-1" }, "tasks[0].id"},
		{"bad task status", func(d *model.Document) { d.Tasks[0].Status = "weird" }, "tasks[0].status"},
		{"task done needs completed", func(d *model.Document) { d.Tasks[0].Status = model.TaskDone }, "tasks[0].completed"},
		{"unknown parent", func(d *model.Document) { d.Tasks[0].Parent = "dm-999" }, "tasks[0].parent"},
		{"self parent", func(d *model.Document) { d.Tasks[0].Parent = "dm-001" }, "tasks[0].parent"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := validDoc()
			tc.mutate(d)
			f := Document(d)
			if !hasField(f, tc.field) {
				t.Fatalf("expected a finding on %q, got: %v", tc.field, f)
			}
		})
	}
}

func TestDuplicateTaskIDs(t *testing.T) {
	d := validDoc()
	d.Tasks = append(d.Tasks, model.Task{ID: "dm-001", Title: "Dup", Status: model.TaskTodo, Created: "2026-07-31"})
	f := Document(d)
	if !hasField(f, "tasks[1].id") {
		t.Fatalf("expected duplicate-id finding, got: %v", f)
	}
}

func TestBlockedNotAllowedOnDone(t *testing.T) {
	d := validDoc()
	d.Tasks[0].Status = model.TaskDone
	d.Tasks[0].Completed = "2026-07-31"
	d.Tasks[0].Blocked = &model.Blocked{Reason: "x", Since: "2026-07-31"}
	f := Document(d)
	if !hasField(f, "tasks[0].blocked") {
		t.Fatalf("expected blocked-not-allowed finding, got: %v", f)
	}
}

func TestParentCycle(t *testing.T) {
	d := validDoc()
	d.Tasks = []model.Task{
		{ID: "dm-001", Title: "A", Status: model.TaskTodo, Created: "2026-07-31", Parent: "dm-002"},
		{ID: "dm-002", Title: "B", Status: model.TaskTodo, Created: "2026-07-31", Parent: "dm-001"},
	}
	f := Document(d)
	found := false
	for _, x := range f {
		if strings.Contains(x.Message, "cycle") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a parent-cycle finding, got: %v", f)
	}
}

func TestBlockerSelfAndUnknown(t *testing.T) {
	d := validDoc()
	d.Tasks[0].Blocked = &model.Blocked{Reason: "x", Since: "2026-07-31", Tasks: []string{"dm-001", "dm-404"}}
	f := Document(d)
	if !hasField(f, "tasks[0].blocked.tasks[0]") || !hasField(f, "tasks[0].blocked.tasks[1]") {
		t.Fatalf("expected self and unknown blocker findings, got: %v", f)
	}
}
