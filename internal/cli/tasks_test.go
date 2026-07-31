package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const projectWithTasks = `schema-version: 1

project:
  id: demo
  title: Demo
  task-id-prefix: dm
  status: in-progress
  created: "2026-07-31"
  started: "2026-07-31"

tasks:
  - id: dm-001
    title: Alpha
    status: in-progress
    created: "2026-07-31"
    started: "2026-07-31"
    tags:
      - research
  - id: dm-002
    title: Beta
    status: todo
    created: "2026-07-31"
    due: "2026-09-01"
  - id: dm-003
    title: Gamma
    status: done
    created: "2026-07-31"
    completed: "2026-07-31"
`

func TestTasksListAllFlagShowsEverything(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), projectWithTasks)

	code, stdout, stderr := run("--root", root, "tasks", "list", "--all")
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	for _, id := range []string{"dm-001", "dm-002", "dm-003"} {
		if !strings.Contains(stdout, id) {
			t.Fatalf("list --all missing %s: %q", id, stdout)
		}
	}
}

func TestTasksListHidesTerminalByDefault(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), projectWithTasks)

	// Default: dm-003 (done) is hidden; open tasks remain.
	_, stdout, _ := run("--root", root, "tasks", "list")
	if !strings.Contains(stdout, "dm-001") || !strings.Contains(stdout, "dm-002") {
		t.Fatalf("open tasks should be listed: %q", stdout)
	}
	if strings.Contains(stdout, "dm-003") {
		t.Fatalf("done task should be hidden by default: %q", stdout)
	}

	// -a reveals it.
	_, allOut, _ := run("--root", root, "tasks", "list", "-a")
	if !strings.Contains(allOut, "dm-003") {
		t.Fatalf("-a should reveal the done task: %q", allOut)
	}

	// An explicit --status done overrides the default and shows only done.
	_, doneOut, _ := run("--root", root, "tasks", "list", "--status", "done")
	if !strings.Contains(doneOut, "dm-003") || strings.Contains(doneOut, "dm-001") {
		t.Fatalf("--status done should show only done: %q", doneOut)
	}
}

func TestTasksListStatusFilterJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), projectWithTasks)

	code, stdout, _ := run("--json", "--root", root, "tasks", "list", "--status", "todo")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var out struct {
		Tasks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(out.Tasks) != 1 || out.Tasks[0].ID != "dm-002" {
		t.Fatalf("status filter got %+v", out.Tasks)
	}
}

const projectWithPriorities = `schema-version: 1

project:
  id: prio
  title: Prio
  task-id-prefix: pr
  status: in-progress
  created: "2026-07-31"
  started: "2026-07-31"

tasks:
  - id: pr-001
    title: No priority
    status: todo
    created: "2026-07-31"
  - id: pr-002
    title: Second
    status: todo
    priority: 2
    created: "2026-07-31"
  - id: pr-003
    title: First
    status: todo
    priority: 1
    created: "2026-07-31"
`

func TestTasksListPriorityOrder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "prio", "prio.tasks.yaml"), projectWithPriorities)

	code, stdout, _ := run("--json", "--root", root, "tasks", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var out struct {
		Tasks []struct {
			ID string `json:"id"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	got := []string{out.Tasks[0].ID, out.Tasks[1].ID, out.Tasks[2].ID}
	want := []string{"pr-003", "pr-002", "pr-001"} // prio 1, prio 2, then unset
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}

	// The text output carries a PRIO column.
	_, text, _ := run("--root", root, "tasks", "list")
	if !strings.Contains(text, "PRIO") {
		t.Fatalf("missing PRIO column: %q", text)
	}
}

func TestTasksShow(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), projectWithTasks)

	code, stdout, _ := run("--root", root, "tasks", "show", "dm-001")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "Alpha") || !strings.Contains(stdout, "research") {
		t.Fatalf("show output missing fields: %q", stdout)
	}
}

func TestTasksShowUnknownIsUsageError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), projectWithTasks)

	code, _, _ := run("--root", root, "tasks", "show", "dm-404")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestProjectsShow(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), projectWithTasks)

	code, stdout, _ := run("--root", root, "projects", "show", "demo")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout, "demo") || !strings.Contains(stdout, "demo/demo.tasks.yaml") {
		t.Fatalf("show output missing id/path: %q", stdout)
	}
	if !strings.Contains(stdout, "1/3 done") {
		t.Fatalf("show output missing progress bar: %q", stdout)
	}
	// Per-status breakdown: one todo, one in-progress, one done.
	for _, want := range []string{"todo:", "in-progress:", "done:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("show output missing status breakdown %q: %q", want, stdout)
		}
	}
}

func TestProjectsShowUnknownIsUsageError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), projectWithTasks)

	code, _, _ := run("--root", root, "projects", "show", "nope")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}
