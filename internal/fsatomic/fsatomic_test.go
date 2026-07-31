package fsatomic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileCreatesAndOverwrites(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")

	if err := WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "hello" {
		t.Fatalf("got %q", b)
	}

	if err := WriteFile(p, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "world" {
		t.Fatalf("got %q", b)
	}

	// No temporary files are left behind.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, found %d: %v", len(entries), entries)
	}
}
