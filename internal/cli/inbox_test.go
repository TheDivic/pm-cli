package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddWithoutProjectCreatesAndUsesTheInbox(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run("--root", root, "tasks", "add", "-t", "call the dentist")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "in-001") {
		t.Fatalf("output: %q", stdout)
	}
	// Creating the inbox is a note on stderr, not a second record on stdout.
	if !strings.Contains(stderr, "created the inbox project") {
		t.Fatalf("stderr should note the inbox creation: %q", stderr)
	}
	if strings.Count(strings.TrimSpace(stdout), "\n") != 0 {
		t.Fatalf("stdout should carry exactly one record: %q", stdout)
	}

	path := filepath.Join(root, "inbox", "inbox.tasks.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("inbox file not created: %v", err)
	}
	if code, _, _ := run("--root", root, "projects", "validate"); code != 0 {
		t.Fatalf("inbox file does not validate")
	}

	// The second capture reuses it rather than creating anything.
	code, stdout, stderr = run("--root", root, "tasks", "add", "-t", "buy milk")
	if code != 0 {
		t.Fatalf("second add exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "in-002") {
		t.Fatalf("second add should continue the sequence: %q", stdout)
	}
	if strings.Contains(stderr, "created the inbox") {
		t.Fatalf("inbox created twice: %q", stderr)
	}
}

func TestInboxProjectShape(t *testing.T) {
	root := t.TempDir()
	run("--root", root, "tasks", "add", "-t", "something")

	_, stdout, _ := run("--json", "--root", root, "projects", "show", "inbox")
	var p struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		TaskIDPrefix string `json:"task_id_prefix"`
		Status       string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &p); err != nil {
		t.Fatalf("show: %v (%q)", err, stdout)
	}
	if p.ID != "inbox" || p.TaskIDPrefix != "in" || p.Title != "Inbox" {
		t.Fatalf("unexpected inbox project: %+v", p)
	}
	// An inbox is continuously worked: never speculative, never finished.
	if p.Status != "in-progress" {
		t.Fatalf("inbox status = %q, want in-progress", p.Status)
	}
}

func TestAddWithProjectStillTargetsIt(t *testing.T) {
	root := seedTwoProjects(t)

	if code, stdout, stderr := run("--root", root, "tasks", "add", "-p", "alpha", "-t", "Explicit"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	} else if !strings.Contains(stdout, "al-003") {
		t.Fatalf("output: %q", stdout)
	}
	// Naming a project must never conjure an inbox.
	if _, err := os.Stat(filepath.Join(root, "inbox")); !os.IsNotExist(err) {
		t.Fatalf("inbox created despite an explicit --project")
	}
}

func TestAddToUnknownProjectStillFails(t *testing.T) {
	root := seedTwoProjects(t)

	code, _, stderr := run("--root", root, "tasks", "add", "-p", "nope", "-t", "Nowhere")
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr, "nope") {
		t.Fatalf("error should name the project: %q", stderr)
	}
}

func TestInboxJSONModeEmitsOnlyTheAddResult(t *testing.T) {
	root := t.TempDir()

	code, stdout, _ := run("--json", "--root", root, "tasks", "add", "-t", "captured")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var r mutationResult
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("stdout must be a single JSON object: %v (%q)", err, stdout)
	}
	if r.Project != "inbox" || r.ID != "in-001" {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestExistingInboxProjectIsReused(t *testing.T) {
	root := t.TempDir()
	// A hand-made inbox at a different path and with a different title is still
	// the inbox: the ID is what identifies it.
	run("--root", root, "projects", "create", "--id", "inbox", "--title", "My Inbox",
		"--task-id-prefix", "in", "--path", filepath.Join(root, "inbox", "inbox.tasks.yaml"))

	code, stdout, stderr := run("--root", root, "tasks", "add", "-t", "captured")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if strings.Contains(stderr, "created the inbox") {
		t.Fatalf("existing inbox should be reused: %q", stderr)
	}
	if !strings.Contains(stdout, "in-001") {
		t.Fatalf("output: %q", stdout)
	}
}

func TestInboxPrefixCollisionIsReported(t *testing.T) {
	root := t.TempDir()
	// Another project already owns the "in" prefix, so the inbox cannot be
	// created; the failure must say why rather than write a broken file.
	run("--root", root, "projects", "create", "--id", "invoices", "--title", "Invoices",
		"--task-id-prefix", "in")

	code, _, stderr := run("--root", root, "tasks", "add", "-t", "captured")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr, "task-id-prefix") {
		t.Fatalf("error should explain the prefix collision: %q", stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "inbox", "inbox.tasks.yaml")); !os.IsNotExist(err) {
		t.Fatalf("a broken inbox file was written")
	}
}
