package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func projectFile(id, prefix string) string {
	return "schema-version: 1\n\nproject:\n  id: " + id +
		"\n  title: " + id + "\n  task-id-prefix: " + prefix +
		"\n  status: idea\n  created: \"2026-07-31\"\n\ntasks: []\n"
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverFindsAndSorts(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "b", "beta.tasks.yaml"), projectFile("beta", "bt"))
	write(t, filepath.Join(root, "a", "alpha.tasks.yaml"), projectFile("alpha", "al"))
	// A task file inside .git must be skipped.
	write(t, filepath.Join(root, ".git", "sneaky.tasks.yaml"), projectFile("sneaky", "sn"))
	// A non-task YAML file must be ignored.
	write(t, filepath.Join(root, "notes.yaml"), "hello: world\n")

	ws, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d: %+v", len(ws.Projects), ws.Projects)
	}
	if ws.Projects[0].ID() != "alpha" || ws.Projects[1].ID() != "beta" {
		t.Fatalf("projects not sorted by ID: %s, %s", ws.Projects[0].ID(), ws.Projects[1].ID())
	}
}

func TestDiscoverDetectsDuplicateID(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "one", "dup.tasks.yaml"), projectFile("dup", "d1"))
	write(t, filepath.Join(root, "two", "dup.tasks.yaml"), projectFile("dup", "d2"))

	ws, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, c := range ws.Conflicts {
		if c.Kind == "project id" && c.Value == "dup" && len(c.Paths) == 2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate project-id conflict, got: %+v", ws.Conflicts)
	}
}

func TestDiscoverFlagsFilenameMismatch(t *testing.T) {
	root := t.TempDir()
	// project id "alpha" but filename stem "wrong".
	write(t, filepath.Join(root, "wrong.tasks.yaml"), projectFile("alpha", "al"))

	ws, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(ws.Projects))
	}
	var found bool
	for _, f := range ws.Projects[0].Findings {
		if f.Field == "project.id" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a filename-stem mismatch finding, got: %+v", ws.Projects[0].Findings)
	}
}

func TestDiscoverRecordsLoadError(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "bad.tasks.yaml"), "schema-version: 1\nproject:\n  id: bad\n")

	ws, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Projects) != 1 || ws.Projects[0].LoadErr == nil {
		t.Fatalf("expected a load error for the malformed file: %+v", ws.Projects)
	}
}
