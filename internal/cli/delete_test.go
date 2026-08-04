package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// treeProject holds a parent with two children and a task blocked by one of
// them, which covers every reference a delete has to reckon with.
const treeProject = `schema-version: 1

project:
  id: demo
  title: Demo
  task-id-prefix: dm
  status: in-progress
  created: "2026-07-31"
  started: "2026-07-31"

tasks:
  - id: dm-001
    title: Parent
    status: todo
    created: "2026-07-31"

  - id: dm-002
    title: Child
    status: todo
    parent: dm-001
    created: "2026-07-31"

  - id: dm-003
    title: Grandchild
    status: todo
    parent: dm-002
    created: "2026-07-31"

  - id: dm-004
    title: Standalone
    status: in-progress
    created: "2026-07-31"
    started: "2026-07-31"
    blocked:
      reason: waiting on the parent
      tasks:
        - dm-001
      since: "2026-07-31"

  - id: dm-005
    title: Unrelated
    status: todo
    created: "2026-07-31"
`

func seedTree(t *testing.T) (root, path string) {
	t.Helper()
	root = t.TempDir()
	path = filepath.Join(root, "demo", "demo.tasks.yaml")
	writeFile(t, path, treeProject)
	return root, path
}

func taskIDs(t *testing.T, root string) []string {
	t.Helper()
	_, stdout, _ := run("--json", "--root", root, "tasks", "list", "--all")
	var out struct {
		Tasks []struct {
			ID string `json:"id"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("list: %v (%q)", err, stdout)
	}
	ids := make([]string, 0, len(out.Tasks))
	for _, task := range out.Tasks {
		ids = append(ids, task.ID)
	}
	// list output is ranked by status, so sort for a stable comparison.
	sort.Strings(ids)
	return ids
}

func TestDeleteUnreferencedTask(t *testing.T) {
	root, _ := seedTree(t)

	code, stdout, stderr := run("--root", root, "tasks", "delete", "dm-005")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "deleted task dm-005") {
		t.Fatalf("output: %q", stdout)
	}
	if got := strings.Join(taskIDs(t, root), ","); strings.Contains(got, "dm-005") {
		t.Fatalf("dm-005 still present: %s", got)
	}
	if code, _, _ := run("--root", root, "projects", "validate"); code != 0 {
		t.Fatalf("file invalid after delete")
	}
}

func TestDeleteRefusesReferencedTask(t *testing.T) {
	root, path := seedTree(t)
	before, _ := os.ReadFile(path)

	code, _, stderr := run("--root", root, "tasks", "delete", "dm-001")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	// The refusal names both the child and the blocker relationship.
	for _, want := range []string{"dm-002", "child of dm-001", "dm-004", "blocked by dm-001", "--cascade"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr missing %q: %s", want, stderr)
		}
	}
	if after, _ := os.ReadFile(path); string(before) != string(after) {
		t.Fatalf("refused delete modified the file")
	}
}

func TestDeleteCascadeRemovesSubtreeAndReferences(t *testing.T) {
	root, _ := seedTree(t)

	code, stdout, stderr := run("--root", root, "tasks", "delete", "dm-001", "--cascade")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	// The whole subtree goes, including the grandchild nobody named.
	for _, id := range []string{"dm-001", "dm-002", "dm-003"} {
		if !strings.Contains(stdout, id) {
			t.Fatalf("output should report %s: %q", id, stdout)
		}
	}
	remaining := strings.Join(taskIDs(t, root), ",")
	if remaining != "dm-004,dm-005" {
		t.Fatalf("remaining tasks = %q, want dm-004,dm-005", remaining)
	}

	// dm-004 keeps its blocking record but loses the dangling reference.
	_, detail, _ := run("--json", "--root", root, "tasks", "show", "dm-004")
	if strings.Contains(detail, "dm-001") {
		t.Fatalf("dm-004 still references the deleted blocker: %s", detail)
	}
	if !strings.Contains(detail, "waiting on the parent") {
		t.Fatalf("dm-004 lost its blocking reason: %s", detail)
	}
	if code, _, _ := run("--root", root, "projects", "validate"); code != 0 {
		t.Fatalf("file invalid after cascade delete")
	}
}

func TestDeleteBatchCoversWholeSubtreeWithoutCascade(t *testing.T) {
	root, _ := seedTree(t)

	// Naming every task in the subtree leaves no dangling parent reference, so
	// the delete is allowed even though dm-004 must still be handled.
	if code, _, stderr := run("--root", root, "tasks", "delete", "dm-002", "dm-003"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if got := strings.Join(taskIDs(t, root), ","); got != "dm-001,dm-004,dm-005" {
		t.Fatalf("remaining = %q", got)
	}
}

func TestDeleteUnknownIDChangesNothing(t *testing.T) {
	root, path := seedTree(t)
	before, _ := os.ReadFile(path)

	code, _, stderr := run("--root", root, "tasks", "delete", "dm-005", "dm-404")
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr, "dm-404") {
		t.Fatalf("stderr should name the unknown ID: %q", stderr)
	}
	if after, _ := os.ReadFile(path); string(before) != string(after) {
		t.Fatalf("file changed despite an unknown ID in the batch")
	}
}

func TestDeleteJSONShapes(t *testing.T) {
	root, _ := seedTree(t)

	// One task: the flat mutation result.
	_, single, _ := run("--json", "--root", root, "tasks", "delete", "dm-005")
	var flat struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal([]byte(single), &flat); err != nil {
		t.Fatalf("not JSON: %v (%q)", err, single)
	}
	if flat.Kind != "deleted task" || flat.ID != "dm-005" {
		t.Fatalf("unexpected single payload: %+v", flat)
	}

	// A cascade that removes more than one: the batch object, listing every
	// task actually deleted rather than only those named.
	_, many, _ := run("--json", "--root", root, "tasks", "delete", "dm-001", "--cascade")
	var batch batchResult
	if err := json.Unmarshal([]byte(many), &batch); err != nil {
		t.Fatalf("not JSON: %v (%q)", err, many)
	}
	if batch.Count != 3 || len(batch.Tasks) != 3 {
		t.Fatalf("want 3 deleted tasks, got %+v", batch)
	}
}

func TestDeleteAcrossProjects(t *testing.T) {
	root := seedTwoProjects(t)

	if code, stdout, stderr := run("--root", root, "tasks", "delete", "al-001", "bt-002"); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	} else if !strings.Contains(stdout, "al-001") || !strings.Contains(stdout, "bt-002") {
		t.Fatalf("output: %q", stdout)
	}
	if got := strings.Join(taskIDs(t, root), ","); got != "al-002,bt-001" {
		t.Fatalf("remaining = %q, want al-002,bt-001", got)
	}
}
