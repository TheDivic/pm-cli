package mutate

import (
	"testing"
	"time"

	"github.com/TheDivic/plaintext-tasks/internal/clock"
	"github.com/TheDivic/plaintext-tasks/internal/model"
)

func fixed() clock.Clock {
	return clock.Fixed{Instant: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)}
}

func baseDoc() *model.Document {
	return &model.Document{
		SchemaVersion: 1,
		Project: model.Project{
			ID: "demo", Title: "Demo", TaskIDPrefix: "dm",
			Status: model.ProjectInProgress, Created: "2026-07-31", Started: "2026-07-31",
		},
		Tasks: []model.Task{
			{ID: "dm-001", Title: "One", Status: model.TaskTodo, Created: "2026-07-31"},
		},
	}
}

func TestNextTaskID(t *testing.T) {
	cases := []struct {
		ids  []string
		want string
	}{
		{nil, "dm-001"},
		{[]string{"dm-001", "dm-005"}, "dm-006"},   // gaps
		{[]string{"dm-001", "dm-1000"}, "dm-1001"}, // more than three digits
	}
	for _, c := range cases {
		d := &model.Document{Project: model.Project{TaskIDPrefix: "dm"}}
		for _, id := range c.ids {
			d.Tasks = append(d.Tasks, model.Task{ID: id})
		}
		if got := NextTaskID(d); got != c.want {
			t.Errorf("NextTaskID(%v) = %s, want %s", c.ids, got, c.want)
		}
	}
}

func TestAddTaskAllocatesAndStamps(t *testing.T) {
	d := baseDoc()
	id, err := AddTask(d, TaskAdd{Title: "Two"}, fixed())
	if err != nil {
		t.Fatal(err)
	}
	if id != "dm-002" {
		t.Fatalf("id = %s", id)
	}
	last := d.Tasks[len(d.Tasks)-1]
	if last.ID != "dm-002" || last.Status != model.TaskTodo || last.Created != "2026-07-31" {
		t.Fatalf("unexpected task: %+v", last)
	}
}

func TestAddTaskInsertsAfterParentSubtree(t *testing.T) {
	d := baseDoc()
	// dm-001 (root); add dm-002 as child of dm-001, then dm-003 root.
	if _, err := AddTask(d, TaskAdd{Title: "child", Parent: "dm-001"}, fixed()); err != nil {
		t.Fatal(err)
	}
	if _, err := AddTask(d, TaskAdd{Title: "root2"}, fixed()); err != nil {
		t.Fatal(err)
	}
	order := []string{d.Tasks[0].ID, d.Tasks[1].ID, d.Tasks[2].ID}
	if order[0] != "dm-001" || order[1] != "dm-002" || order[2] != "dm-003" {
		t.Fatalf("order = %v; child should follow its parent", order)
	}
}

func TestProjectStatusManagesDates(t *testing.T) {
	d := baseDoc()
	d.Project.Status = model.ProjectIdea
	d.Project.Started = ""

	if err := ProjectStatus(d, model.ProjectInProgress, "", fixed()); err != nil {
		t.Fatal(err)
	}
	if d.Project.Started != "2026-07-31" {
		t.Fatalf("started not set: %q", d.Project.Started)
	}
	// blocked requires a reason.
	if err := ProjectStatus(d, model.ProjectBlocked, "", fixed()); err == nil {
		t.Fatal("expected error without reason")
	}
	if err := ProjectStatus(d, model.ProjectBlocked, "paused", fixed()); err != nil {
		t.Fatal(err)
	}
	if d.Project.Blocked == nil {
		t.Fatal("blocked not set")
	}
	// leaving blocked removes the record.
	if err := ProjectStatus(d, model.ProjectInProgress, "", fixed()); err != nil {
		t.Fatal(err)
	}
	if d.Project.Blocked != nil {
		t.Fatal("blocked should be cleared")
	}
}

func TestTaskStatusTerminalFields(t *testing.T) {
	d := baseDoc()
	if err := TaskStatus(d, "dm-001", model.TaskDone, "", fixed()); err != nil {
		t.Fatal(err)
	}
	if d.Tasks[0].Completed != "2026-07-31" {
		t.Fatalf("completed not set: %+v", d.Tasks[0])
	}
	// Moving back to todo clears completed so validation stays consistent.
	if err := TaskStatus(d, "dm-001", model.TaskTodo, "", fixed()); err != nil {
		t.Fatal(err)
	}
	if d.Tasks[0].Completed != "" {
		t.Fatalf("completed should be cleared: %+v", d.Tasks[0])
	}
	// cancelled requires a reason.
	if err := TaskStatus(d, "dm-001", model.TaskCancelled, "", fixed()); err == nil {
		t.Fatal("expected error without reason")
	}
}

func TestBlockAndUnblock(t *testing.T) {
	d := baseDoc()
	if err := BlockTask(d, "dm-001", "waiting", nil, fixed()); err != nil {
		t.Fatal(err)
	}
	if d.Tasks[0].Blocked == nil || d.Tasks[0].Status != model.TaskTodo {
		t.Fatalf("block did not record or changed status: %+v", d.Tasks[0])
	}
	if err := UnblockTask(d, "dm-001"); err != nil {
		t.Fatal(err)
	}
	if d.Tasks[0].Blocked != nil {
		t.Fatal("blocked not cleared")
	}
	// Cannot block a done task.
	if err := TaskStatus(d, "dm-001", model.TaskDone, "", fixed()); err != nil {
		t.Fatal(err)
	}
	if err := BlockTask(d, "dm-001", "nope", nil, fixed()); err == nil {
		t.Fatal("expected error blocking a done task")
	}
}

func TestEditTaskSetAndClear(t *testing.T) {
	d := baseDoc()
	p := 2
	if err := EditTask(d, "dm-001", TaskEdit{Title: ptrString("Renamed"), Priority: &p, AddTags: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	if d.Tasks[0].Title != "Renamed" || d.Tasks[0].Priority == nil || *d.Tasks[0].Priority != 2 {
		t.Fatalf("edit failed: %+v", d.Tasks[0])
	}
	if err := EditTask(d, "dm-001", TaskEdit{ClearPriority: true, RemoveTags: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	if d.Tasks[0].Priority != nil || len(d.Tasks[0].Tags) != 0 {
		t.Fatalf("clear failed: %+v", d.Tasks[0])
	}
}

func ptrString(s string) *string { return &s }
