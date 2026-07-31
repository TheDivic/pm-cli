package lockfile

import (
	"path/filepath"
	"testing"
)

func TestAcquireReleaseAndReacquire(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.tasks.yaml")

	release, err := Acquire(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}

	// The same path can be locked again once released.
	release2, err := Acquire(p)
	if err != nil {
		t.Fatalf("re-acquire: %v", err)
	}
	if err := release2(); err != nil {
		t.Fatal(err)
	}
}

func TestDifferentPathsLockIndependently(t *testing.T) {
	dir := t.TempDir()
	r1, err := Acquire(filepath.Join(dir, "a.tasks.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r1() }()
	r2, err := Acquire(filepath.Join(dir, "b.tasks.yaml"))
	if err != nil {
		t.Fatalf("second path should lock independently: %v", err)
	}
	if err := r2(); err != nil {
		t.Fatal(err)
	}
}
