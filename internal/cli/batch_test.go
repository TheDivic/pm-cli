package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedTwoProjects creates two projects with two tasks each so batch commands can
// be exercised both within one file and across files.
func seedTwoProjects(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, p := range []struct{ id, prefix string }{{"alpha", "al"}, {"beta", "bt"}} {
		if code, _, stderr := run("--root", root, "projects", "create",
			"--id", p.id, "--title", strings.ToUpper(p.id), "--task-id-prefix", p.prefix,
			"--status", "in-progress"); code != 0 {
			t.Fatalf("create %s: exit %d: %s", p.id, code, stderr)
		}
		for _, title := range []string{"One", "Two"} {
			if code, _, stderr := run("--root", root, "tasks", "add", "-p", p.id, "-t", title, "-s", "todo"); code != 0 {
				t.Fatalf("add to %s: exit %d: %s", p.id, code, stderr)
			}
		}
	}
	return root
}

func taskStatus(t *testing.T, root, id string) string {
	t.Helper()
	code, stdout, stderr := run("--json", "--root", root, "tasks", "show", id)
	if code != 0 {
		t.Fatalf("show %s: exit %d: %s", id, code, stderr)
	}
	var d struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &d); err != nil {
		t.Fatalf("show %s: %v", id, err)
	}
	return d.Status
}

func TestBatchStatusWithinOneFile(t *testing.T) {
	root := seedTwoProjects(t)

	code, stdout, stderr := run("--root", root, "tasks", "status", "al-001", "al-002", "in-progress")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	// One human-readable line per changed task.
	if n := strings.Count(strings.TrimSpace(stdout), "\n") + 1; n != 2 {
		t.Fatalf("want 2 output lines, got %d: %q", n, stdout)
	}
	for _, id := range []string{"al-001", "al-002"} {
		if got := taskStatus(t, root, id); got != "in-progress" {
			t.Fatalf("%s status = %q, want in-progress", id, got)
		}
	}
}

func TestBatchStatusAcrossProjects(t *testing.T) {
	root := seedTwoProjects(t)

	if code, _, stderr := run("--root", root, "tasks", "status", "al-001", "bt-002", "done"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	for _, id := range []string{"al-001", "bt-002"} {
		if got := taskStatus(t, root, id); got != "done" {
			t.Fatalf("%s status = %q, want done", id, got)
		}
	}
	// The untouched tasks keep their original status.
	if got := taskStatus(t, root, "al-002"); got != "todo" {
		t.Fatalf("al-002 status = %q, want todo", got)
	}
}

func TestBatchStatusSingleIDKeepsFlatJSON(t *testing.T) {
	root := seedTwoProjects(t)

	_, stdout, _ := run("--json", "--root", root, "tasks", "status", "al-001", "done")
	var flat struct {
		Kind   string `json:"kind"`
		ID     string `json:"id"`
		Status string `json:"status"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &flat); err != nil {
		t.Fatalf("not JSON: %v (%q)", err, stdout)
	}
	if flat.ID != "al-001" || flat.Status != "done" || flat.Path == "" {
		t.Fatalf("single-task JSON changed shape: %+v", flat)
	}
}

func TestBatchStatusMultipleIDsUsesBatchJSON(t *testing.T) {
	root := seedTwoProjects(t)

	_, stdout, _ := run("--json", "--root", root, "tasks", "status", "al-001", "bt-001", "done")
	var batch batchResult
	if err := json.Unmarshal([]byte(stdout), &batch); err != nil {
		t.Fatalf("not JSON: %v (%q)", err, stdout)
	}
	if batch.Count != 2 || len(batch.Tasks) != 2 {
		t.Fatalf("want 2 tasks, got %+v", batch)
	}
	if batch.Status != "done" || batch.Tasks[0].ID != "al-001" || batch.Tasks[1].ID != "bt-001" {
		t.Fatalf("unexpected batch payload: %+v", batch)
	}
}

func TestBatchUnknownIDChangesNothing(t *testing.T) {
	root := seedTwoProjects(t)
	path := filepath.Join(root, "alpha", "alpha.tasks.yaml")
	before, _ := os.ReadFile(path)

	code, _, stderr := run("--root", root, "tasks", "status", "al-001", "al-404", "done")
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr, "al-404") {
		t.Fatalf("error should name the unknown ID: %q", stderr)
	}
	// Resolution happens before any write, so the valid task is untouched too.
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("file changed despite an unresolvable ID in the batch")
	}
	if got := taskStatus(t, root, "al-001"); got != "todo" {
		t.Fatalf("al-001 status = %q, want todo", got)
	}
}

func TestBatchFailureLeavesItsFileUnchanged(t *testing.T) {
	root := seedTwoProjects(t)
	path := filepath.Join(root, "alpha", "alpha.tasks.yaml")
	before, _ := os.ReadFile(path)

	// Cancelling requires a reason, so the second task fails and the whole file
	// must stay as it was.
	code, _, stderr := run("--root", root, "tasks", "status", "al-001", "al-002", "cancelled")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr, "al-001") {
		t.Fatalf("error should name the failing task: %q", stderr)
	}
	if after, _ := os.ReadFile(path); string(before) != string(after) {
		t.Fatalf("failed batch modified the file")
	}
}

func TestBatchReportsTasksAlreadyCommitted(t *testing.T) {
	root := seedTwoProjects(t)

	// alpha's tasks succeed; beta's second task cannot be unblocked because it
	// was never blocked, so the command fails after alpha was already written.
	run("--root", root, "tasks", "block", "al-001", "-r", "waiting")
	code, _, stderr := run("--root", root, "tasks", "unblock", "al-001", "bt-001")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr, "already updated al-001") {
		t.Fatalf("stderr should name the committed tasks: %q", stderr)
	}
}

func TestBatchEditAndUnblock(t *testing.T) {
	root := seedTwoProjects(t)

	if code, _, stderr := run("--root", root, "tasks", "edit", "al-001", "al-002", "bt-001",
		"--add-tag", "sweep", "--priority", "2"); code != 0 {
		t.Fatalf("batch edit exit %d: %s", code, stderr)
	}
	for _, id := range []string{"al-001", "al-002", "bt-001"} {
		_, stdout, _ := run("--json", "--root", root, "tasks", "show", id)
		if !strings.Contains(stdout, `"sweep"`) || !strings.Contains(stdout, `"priority": 2`) {
			t.Fatalf("%s missing the batch edit: %s", id, stdout)
		}
	}

	// Blocking and unblocking accept several IDs too.
	if code, _, stderr := run("--root", root, "tasks", "block", "al-001", "bt-001", "-r", "waiting on design"); code != 0 {
		t.Fatalf("batch block exit %d: %s", code, stderr)
	}
	if code, _, stderr := run("--root", root, "tasks", "unblock", "al-001", "bt-001"); code != 0 {
		t.Fatalf("batch unblock exit %d: %s", code, stderr)
	}
}

func TestBatchEditRefusesSharedTitle(t *testing.T) {
	root := seedTwoProjects(t)

	code, _, stderr := run("--root", root, "tasks", "edit", "al-001", "al-002", "--title", "Same")
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr, "--title") {
		t.Fatalf("error should explain the --title restriction: %q", stderr)
	}
}

func TestBatchRepeatedIDAppliesOnce(t *testing.T) {
	root := seedTwoProjects(t)

	code, stdout, stderr := run("--root", root, "tasks", "status", "al-001", "al-001", "done")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if strings.Count(stdout, "al-001") != 1 {
		t.Fatalf("a repeated ID should be reported once: %q", stdout)
	}
}

func TestStatusStillRequiresAStatusArgument(t *testing.T) {
	root := seedTwoProjects(t)

	if code, _, _ := run("--root", root, "tasks", "status", "al-001"); code != 2 {
		t.Fatalf("missing status: exit %d, want 2", code)
	}
	// A forgotten status leaves the last task ID in the status position, which
	// reports as an invalid status rather than silently doing nothing.
	code, _, stderr := run("--root", root, "tasks", "status", "al-001", "al-002")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr, "al-002") {
		t.Fatalf("error should quote the bad status: %q", stderr)
	}
}
