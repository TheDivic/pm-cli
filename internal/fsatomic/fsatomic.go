// Package fsatomic writes files atomically: content goes to a temporary file in
// the same directory, is flushed, then renamed over the target so a failure
// before the rename leaves the original bytes unchanged.
//
// File locking against concurrent commands in one checkout is a separate
// concern added with the mutation commands.
package fsatomic

import (
	"os"
	"path/filepath"
)

// WriteFile atomically writes data to path with the given permissions.
func WriteFile(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".pm-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Clean up the temp file unless the rename below consumes it.
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
