package query

import (
	"testing"

	"github.com/TheDivic/plaintext-tasks/internal/discover"
	"github.com/TheDivic/plaintext-tasks/internal/model"
)

func demoProject() *model.Project {
	return &model.Project{ID: "demo", Areas: []string{"knowledge-work"}}
}

func mk(id, status, parent, due string, tags []string, blocked bool) TaskRef {
	t := &model.Task{ID: id, Status: model.TaskStatus(status), Parent: parent, Due: due, Tags: tags}
	if blocked {
		t.Blocked = &model.Blocked{Reason: "x", Since: "2026-07-31"}
	}
	return TaskRef{Task: t, Project: demoProject(), Path: "demo.tasks.yaml"}
}

func ids(refs []TaskRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Task.ID)
	}
	return out
}

func TestFilterByStatusAndTag(t *testing.T) {
	refs := []TaskRef{
		mk("dm-001", "todo", "", "", []string{"research"}, false),
		mk("dm-002", "done", "", "", []string{"research"}, false),
		mk("dm-003", "todo", "", "", []string{"docs"}, false),
	}
	got := ids(Filter(refs, TaskFilter{Statuses: []string{"todo"}, Tags: []string{"research"}}))
	if len(got) != 1 || got[0] != "dm-001" {
		t.Fatalf("got %v, want [dm-001]", got)
	}
}

func TestFilterOrWithinCategory(t *testing.T) {
	refs := []TaskRef{
		mk("dm-001", "todo", "", "", nil, false),
		mk("dm-002", "done", "", "", nil, false),
		mk("dm-003", "backlog", "", "", nil, false),
	}
	got := ids(Filter(refs, TaskFilter{Statuses: []string{"todo", "done"}}))
	if len(got) != 2 {
		t.Fatalf("got %v, want two matches", got)
	}
}

func TestFilterBlocked(t *testing.T) {
	refs := []TaskRef{
		mk("dm-001", "todo", "", "", nil, true),
		mk("dm-002", "todo", "", "", nil, false),
	}
	yes := true
	got := ids(Filter(refs, TaskFilter{Blocked: &yes}))
	if len(got) != 1 || got[0] != "dm-001" {
		t.Fatalf("blocked filter got %v", got)
	}
}

func TestFilterParentAndDue(t *testing.T) {
	refs := []TaskRef{
		mk("dm-001", "todo", "dm-000", "2026-07-10", nil, false),
		mk("dm-002", "todo", "", "2026-08-01", nil, false),
		mk("dm-003", "todo", "dm-000", "", nil, false),
	}
	byParent := ids(Filter(refs, TaskFilter{Parent: "dm-000", HasParent: true}))
	if len(byParent) != 2 {
		t.Fatalf("parent filter got %v", byParent)
	}
	before := ids(Filter(refs, TaskFilter{DueBefore: "2026-07-31"}))
	if len(before) != 1 || before[0] != "dm-001" {
		t.Fatalf("due-before got %v", before)
	}
	on := ids(Filter(refs, TaskFilter{DueOn: "2026-08-01"}))
	if len(on) != 1 || on[0] != "dm-002" {
		t.Fatalf("due-on got %v", on)
	}
}

func TestFilterByArea(t *testing.T) {
	refs := []TaskRef{mk("dm-001", "todo", "", "", nil, false)}
	if got := Filter(refs, TaskFilter{Areas: []string{"nope"}}); len(got) != 0 {
		t.Fatalf("expected no matches for missing area, got %v", ids(got))
	}
	if got := Filter(refs, TaskFilter{Areas: []string{"knowledge-work"}}); len(got) != 1 {
		t.Fatalf("expected a match for the project area, got %v", ids(got))
	}
}

func TestFlattenPreservesOrder(t *testing.T) {
	ws := &discover.Workspace{Projects: []discover.Project{
		{Path: "a.tasks.yaml", Doc: &model.Document{
			Project: model.Project{ID: "a"},
			Tasks:   []model.Task{{ID: "a-001"}, {ID: "a-002"}},
		}},
		{Path: "b.tasks.yaml", LoadErr: errString("boom")}, // skipped
	}}
	got := ids(Flatten(ws))
	if len(got) != 2 || got[0] != "a-001" || got[1] != "a-002" {
		t.Fatalf("flatten got %v", got)
	}
}

func TestSortForList(t *testing.T) {
	refs := []TaskRef{
		mk("dm-001", "todo", "", "", nil, false),
		mk("dm-002", "done", "", "", nil, false),
		mk("dm-003", "in-progress", "", "", nil, false),
		mk("dm-004", "todo", "", "", nil, false),
		mk("dm-005", "in-review", "", "", nil, false),
		mk("dm-006", "backlog", "", "", nil, false),
	}
	SortForList(refs)
	got := ids(refs)
	// in-review, in-progress, then todo (file order dm-001 before dm-004),
	// backlog, done.
	want := []string{"dm-005", "dm-003", "dm-001", "dm-004", "dm-006", "dm-002"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestSortForListPriority(t *testing.T) {
	p := func(n int) *int { return &n }
	refs := []TaskRef{
		{Task: &model.Task{ID: "dm-001", Status: model.TaskTodo}, Project: demoProject()},
		{Task: &model.Task{ID: "dm-002", Status: model.TaskTodo, Priority: p(2)}, Project: demoProject()},
		{Task: &model.Task{ID: "dm-003", Status: model.TaskTodo, Priority: p(1)}, Project: demoProject()},
		{Task: &model.Task{ID: "dm-004", Status: model.TaskInProgress}, Project: demoProject()},
	}
	SortForList(refs)
	got := ids(refs)
	// in-progress first, then todo by priority (1, 2, then unset last).
	want := []string{"dm-004", "dm-003", "dm-002", "dm-001"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestFind(t *testing.T) {
	refs := []TaskRef{mk("dm-001", "todo", "", "", nil, false)}
	if Find(refs, "dm-001") == nil {
		t.Fatal("expected to find dm-001")
	}
	if Find(refs, "dm-999") != nil {
		t.Fatal("did not expect to find dm-999")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
