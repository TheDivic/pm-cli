package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nonCanonical is valid but not canonically formatted (no blank lines between
// the top-level sections).
const nonCanonical = `schema-version: 1
project:
  id: demo
  title: Demo
  task-id-prefix: dm
  status: backlog
  created: "2026-07-31"
tasks: []
`

func TestProjectsFormatRewritesAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "demo", "demo.tasks.yaml")
	writeFile(t, path, nonCanonical)

	code, stdout, stderr := run("--root", root, "projects", "format", "--all")
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "formatted") {
		t.Fatalf("expected a formatted report: %q", stdout)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\n\ntasks:") {
		t.Fatalf("file was not canonicalized:\n%s", data)
	}

	// Running again makes no change.
	code2, stdout2, _ := run("--root", root, "projects", "format", "--all")
	if code2 != 0 || !strings.Contains(stdout2, "unchanged") {
		t.Fatalf("second run: exit %d, out %q", code2, stdout2)
	}
}

func TestProjectsFormatRejectsInvalidAndLeavesFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "broken", "broken.tasks.yaml")
	writeFile(t, path, invalidProject)
	before, _ := os.ReadFile(path)

	code, _, _ := run("--root", root, "projects", "format", "--all")
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("invalid file must be left unchanged")
	}
}

func TestProjectsFormatRequiresExplicitTarget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), validProject)

	code, _, _ := run("--root", root, "projects", "format")
	if code != 2 {
		t.Fatalf("no target: exit = %d, want 2", code)
	}
	code, _, _ = run("--root", root, "projects", "format", "demo", "--all")
	if code != 2 {
		t.Fatalf("both id and --all: exit = %d, want 2", code)
	}
}
