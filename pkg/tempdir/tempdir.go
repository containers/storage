package tempdir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/storage/pkg/lockfile"
	"github.com/sirupsen/logrus"
)

// TODO: implement alternative for: system.EnsureRemoveAll(dir)

// TempDir represents a temporary directory that is created in a specified root directory.
type TempDir struct {
	rootDir      string
	tempDirPath  string
	lockFilePath string
	lock         *lockfile.LockFile
}

// recover checks for stale temporary directories in the specified root directory.
// If any stale directories are found, they are removed.
// Stale directories are identified by their naming pattern "temp-dir-*",
// and the function attempts to acquire a lock on each directory before
// removing it. If a lock cannot be acquired, the directory is skipped, because other
// processes may still be using it.
//
// It is important to note that this function should be called before
// creating a new temporary directory to ensure a clean state.
//
// Required global storage lock must be held before calling this function.
func recover(rootDir string) error {
	tempDirPattern := filepath.Join(rootDir, "temp-dir-*")
	tempDirs, err := filepath.Glob(tempDirPattern)
	if err != nil {
		return fmt.Errorf("error finding tmp dirs in %s: %w", rootDir, err)
	}

	var recoveryErrors []error

	for _, tempDirPath := range tempDirs {
		lockFilePath := filepath.Join(tempDirPath, ".lock")
		lock, err := lockfile.GetLockFile(lockFilePath)
		if err != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("error getting lock file %s: %w", lockFilePath, err))
			continue
		}

		if err := lock.TryLock(); err == nil {
			lock.Unlock()
			if rmErr := os.RemoveAll(tempDirPath); rmErr != nil && !os.IsNotExist(rmErr) {
				recoveryErrors = append(recoveryErrors, fmt.Errorf("error removing stale temp dir %s: %w", tempDirPath, rmErr))
			}
		} else {
			logrus.Debugf("Skipping cleanup for locked directory corresponding to %s", lockFilePath)
		}
	}

	return errors.Join(recoveryErrors...)
}

// NewTempDir creates a new temporary directory in the specified root directory.
// Root directory must exist and be writable. Root directory is cleared of stale
// temporary directories and lock files before creating a new temporary directory.
//
// Required global storage lock must be held before calling this function.
func NewTempDir(rootDir string) (*TempDir, error) {
	if err := recover(rootDir); err != nil {
		return nil, fmt.Errorf("recovering from stale temp dirs failed: %w", err)
	}

	pid := os.Getpid()

	tempDirPath := filepath.Join(rootDir, fmt.Sprintf("temp-dir-%d", pid))
	if err := os.MkdirAll(tempDirPath, 0o755); err != nil {
		return nil, fmt.Errorf("creating .trash failed: %w", err)
	}

	lockFilePath := filepath.Join(tempDirPath, ".lock")
	lock, err := lockfile.GetLockFile(lockFilePath)
	if err != nil {
		return nil, err
	}
	if err := lock.TryLock(); err != nil {
		return nil, fmt.Errorf("locking temp dir failed: %w", err)
	}

	td := &TempDir{
		rootDir:      rootDir,
		tempDirPath:  tempDirPath,
		lockFilePath: lockFilePath,
		lock:         lock,
	}
	return td, nil
}

// Add moves the specified file to the temporary directory.
func (td *TempDir) Add(path string, prefix string) error {
	td.lock.AssertLocked()

	base := filepath.Base(path)

	if prefix != "" {
		base = prefix + "-" + base
	}
	dest := filepath.Join(td.tempDirPath, base)
	return os.Rename(path, dest)
}

// Note: Cleanup must be deferred before differing unlocking global storage locks
// to be executed after the global storage lock is released to benefit from
// not performing slow removal files from disk during global lock.
func (td *TempDir) Cleanup() (err error) {
	td.lock.Unlock()

	if err := os.RemoveAll(td.tempDirPath); err != nil {
		return fmt.Errorf("removing temp dir failed: %w", err)
	}
	return nil
}

func (td *TempDir) Path() string {
	return td.tempDirPath
}
