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

func TestTasksListAll(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), projectWithTasks)

	code, stdout, stderr := run("--root", root, "tasks", "list")
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	for _, id := range []string{"dm-001", "dm-002", "dm-003"} {
		if !strings.Contains(stdout, id) {
			t.Fatalf("list missing %s: %q", id, stdout)
		}
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
	if !strings.Contains(stdout, "3 total") {
		t.Fatalf("show output missing task count: %q", stdout)
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
