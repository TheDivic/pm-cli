package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validProject = `schema-version: 1

project:
  id: demo
  title: Demo Project
  task-id-prefix: dm
  status: in-progress
  created: "2026-07-31"
  started: "2026-07-31"

tasks:
  - id: dm-001
    title: First
    status: todo
    created: "2026-07-31"
`

const invalidProject = `schema-version: 1

project:
  id: broken
  title: Broken
  task-id-prefix: bk
  status: nonsense
  created: "2026-07-31"

tasks: []
`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProjectsListText(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, stdout, stderr := run("--root", root, "projects", "list")
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "demo") || !strings.Contains(stdout, "Demo Project") {
		t.Fatalf("list output missing project: %q", stdout)
	}
}

func TestProjectsListJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, stdout, _ := run("--json", "--root", root, "projects", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var out struct {
		Projects []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"projects"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(out.Projects) != 1 || out.Projects[0].ID != "demo" {
		t.Fatalf("unexpected projects: %+v", out.Projects)
	}
}

func listProjectFile(id, prio, created string) string {
	p := "schema-version: 1\n\nproject:\n  id: " + id + "\n  title: " + id + "\n  task-id-prefix: " + id[:1]
	p += "\n  status: idea\n"
	if prio != "" {
		p += "  priority: " + prio + "\n"
	}
	p += "  created: \"" + created + "\"\n\ntasks: []\n"
	return p
}

func TestProjectsListSortOrder(t *testing.T) {
	root := t.TempDir()
	// Same priority, different created; a lower priority; and no priority.
	writeFile(t, filepath.Join(root, "a.tasks.yaml"), listProjectFile("a", "1", "2026-02-01"))
	writeFile(t, filepath.Join(root, "c.tasks.yaml"), listProjectFile("c", "1", "2026-01-01"))
	writeFile(t, filepath.Join(root, "b.tasks.yaml"), listProjectFile("b", "2", "2026-01-01"))
	writeFile(t, filepath.Join(root, "z.tasks.yaml"), listProjectFile("z", "", "2026-01-01"))

	code, stdout, _ := run("--json", "--root", root, "projects", "list")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var out struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	var got []string
	for _, p := range out.Projects {
		got = append(got, p.ID)
	}
	// c (p1, Jan) < a (p1, Feb) < b (p2) < z (no priority).
	want := []string{"c", "a", "b", "z"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestProjectsValidateValid(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, stdout, _ := run("--root", root, "projects", "validate")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "valid") {
		t.Fatalf("unexpected output: %q", stdout)
	}
}

func TestProjectsValidateInvalidExitsOne(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "broken", "broken.tasks.yaml"), invalidProject)

	code, stdout, _ := run("--root", root, "projects", "validate")
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stdout, "status") {
		t.Fatalf("expected a status finding in output: %q", stdout)
	}
}

func TestProjectsValidateUnknownIDIsUsageError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, _, stderr := run("--root", root, "projects", "validate", "nope")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (stderr: %s)", code, stderr)
	}
}
