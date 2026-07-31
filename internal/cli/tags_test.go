package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const taggedProject = `schema-version: 1

project:
  id: demo
  title: Demo
  task-id-prefix: dm
  status: in-progress
  created: "2026-07-31"
  started: "2026-07-31"

tasks:
  - id: dm-001
    title: One
    status: todo
    created: "2026-07-31"
    tags:
      - alpha
      - beta
  - id: dm-002
    title: Two
    status: done
    created: "2026-07-31"
    completed: "2026-07-31"
    tags:
      - alpha
`

func TestTagsCommand(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), taggedProject)

	code, stdout, stderr := run("--root", root, "tags")
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	// alpha (2) should be listed before beta (1) since more-used tags come first.
	ai := strings.Index(stdout, "alpha")
	bi := strings.Index(stdout, "beta")
	if ai < 0 || bi < 0 || ai > bi {
		t.Fatalf("unexpected tag order/content: %q", stdout)
	}
}

func TestTagsCommandJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "demo", "demo.tasks.yaml"), taggedProject)

	code, stdout, _ := run("--json", "--root", root, "tags")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var out struct {
		Tags []struct {
			Tag   string `json:"tag"`
			Count int    `json:"count"`
		} `json:"tags"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	got := map[string]int{}
	for _, tc := range out.Tags {
		got[tc.Tag] = tc.Count
	}
	// alpha is on both tasks (including the done one); beta on one.
	if got["alpha"] != 2 || got["beta"] != 1 {
		t.Fatalf("unexpected counts: %+v", got)
	}
}
