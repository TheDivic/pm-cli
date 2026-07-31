// Package lockfile provides an advisory exclusive lock for a task file so that
// concurrent pm commands in one checkout cannot allocate the same task ID or
// overwrite one another. The lock is held on a sidecar file in the temp
// directory keyed by the target's absolute path, so it survives the atomic
// rename of the target itself.
//
// This uses flock and therefore targets Unix (Linux and macOS). Git remains
// responsible for changes made in separate clones or sessions.
package lockfile

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"syscall"
)

// Acquire takes an exclusive lock for targetPath and returns a release func.
func Acquire(targetPath string) (release func() error, err error) {
	abs, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(abs))
	dir := filepath.Join(os.TempDir(), "pm-locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(dir, hex.EncodeToString(sum[:])+".lock")

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() error {
		unlockErr := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		closeErr := f.Close()
		if unlockErr != nil {
			return unlockErr
		}
		return closeErr
	}, nil
}
