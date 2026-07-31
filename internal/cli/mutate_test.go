package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAddStatusLifecycle(t *testing.T) {
	root := t.TempDir()

	if code, _, stderr := run("--root", root, "projects", "create",
		"--id", "demo", "--title", "Demo", "--task-id-prefix", "dm", "--status", "in-progress"); code != 0 {
		t.Fatalf("create exit %d: %s", code, stderr)
	}
	path := filepath.Join(root, "demo", "demo.tasks.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	if code, stdout, stderr := run("--root", root, "tasks", "add", "--project", "demo", "--title", "First"); code != 0 {
		t.Fatalf("add exit %d: %s", code, stderr)
	} else if !strings.Contains(stdout, "dm-001") {
		t.Fatalf("add output: %q", stdout)
	}

	if code, _, stderr := run("--root", root, "tasks", "status", "dm-001", "in-progress"); code != 0 {
		t.Fatalf("status exit %d: %s", code, stderr)
	}

	// The file remains valid after the mutations.
	if code, _, _ := run("--root", root, "projects", "validate"); code != 0 {
		t.Fatalf("validate exit %d", code)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "started: \"") {
		t.Fatalf("started date not stamped:\n%s", data)
	}
}

func TestCreateRefusesDuplicateID(t *testing.T) {
	root := t.TempDir()
	run("--root", root, "projects", "create", "--id", "demo", "--title", "Demo", "--task-id-prefix", "dm")
	code, _, _ := run("--root", root, "projects", "create", "--id", "demo", "--title", "Again", "--task-id-prefix", "xx")
	if code != 1 {
		t.Fatalf("duplicate id: exit %d, want 1", code)
	}
}

func TestBlockRequiresReason(t *testing.T) {
	root := t.TempDir()
	run("--root", root, "projects", "create", "--id", "demo", "--title", "Demo", "--task-id-prefix", "dm")
	run("--root", root, "tasks", "add", "--project", "demo", "--title", "T")
	code, _, _ := run("--root", root, "tasks", "block", "dm-001") // missing --reason
	if code != 2 {
		t.Fatalf("block without reason: exit %d, want 2", code)
	}
}

func TestStatusUnknownTaskIsUsageError(t *testing.T) {
	root := t.TempDir()
	run("--root", root, "projects", "create", "--id", "demo", "--title", "Demo", "--task-id-prefix", "dm")
	code, _, _ := run("--root", root, "tasks", "status", "dm-404", "done")
	if code != 2 {
		t.Fatalf("unknown task: exit %d, want 2", code)
	}
}

func TestFailedMutationLeavesFileUnchanged(t *testing.T) {
	root := t.TempDir()
	run("--root", root, "projects", "create", "--id", "demo", "--title", "Demo", "--task-id-prefix", "dm")
	run("--root", root, "tasks", "add", "--project", "demo", "--title", "T")
	path := filepath.Join(root, "demo", "demo.tasks.yaml")
	before, _ := os.ReadFile(path)

	// priority 0 is invalid; the post-mutation validation must reject it.
	code, _, _ := run("--root", root, "tasks", "edit", "dm-001", "--priority", "0")
	if code != 1 {
		t.Fatalf("invalid edit: exit %d, want 1", code)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("file must be unchanged after a failed mutation")
	}
}

func TestTasksAddDescriptionFromStdin(t *testing.T) {
	root := t.TempDir()
	run("--root", root, "projects", "create", "--id", "demo", "--title", "Demo", "--task-id-prefix", "dm")
	// run() uses an empty stdin; here we only assert the flag path works with a file.
	descPath := filepath.Join(t.TempDir(), "desc.md")
	if err := os.WriteFile(descPath, []byte("Scope and acceptance.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := run("--root", root, "tasks", "add", "--project", "demo", "--title", "T", "--description-file", descPath)
	if code != 0 {
		t.Fatalf("add with description: exit %d: %s", code, stderr)
	}
	data, _ := os.ReadFile(filepath.Join(root, "demo", "demo.tasks.yaml"))
	if !strings.Contains(string(data), "description: |") || !strings.Contains(string(data), "Scope and acceptance.") {
		t.Fatalf("description not written:\n%s", data)
	}
}
