package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatcher(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("build/\n*.tmp\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		path  string
		isDir bool
		want  bool
	}{
		{"build", true, true},
		{"scratch.tmp", false, true},
		{"src/main.go", false, false},
		{"notes.md", false, false},
	}
	for _, c := range cases {
		if got := m.Ignored(c.path, c.isDir); got != c.want {
			t.Errorf("Ignored(%q, %v) = %v, want %v", c.path, c.isDir, got, c.want)
		}
	}
}

func TestNilMatcherIgnoresNothing(t *testing.T) {
	var m *Matcher
	if m.Ignored("anything", false) || m.Ignored("dir", true) {
		t.Fatal("nil matcher must ignore nothing")
	}
}
