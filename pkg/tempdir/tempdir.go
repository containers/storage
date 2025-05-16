package tempdir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/storage/pkg/lockfile"
	"github.com/sirupsen/logrus"
)

// TODO: implement alternative for: system.EnsureRemoveAll(dir)

/*
Locking rules and invariants for TempDir and its recovery mechanism:

1. TempDir Instance Locks:
  - Path: 'RootDir/lock-XYZ' (in the root directory)
  - Each TempDir instance, when its specific temporary directory is first initialized
    (via initInstanceTempDir, typically called by Add), acquires and holds an
    exclusive lock on this file.
  - This lock signifies that the temporary directory is in active use by the
    process/goroutine that holds the TempDir object.

2. Stale Directory Recovery (initiated by NewTempDir):
  - During RecoverStaleDirs, the code attempts to identify and clean up stale
    temporary directories.
  - For each potential stale directory (found by listPotentialStaleDirs), it
    attempts to TryLock() its instance lock file.
  - If TryLock() succeeds: The directory is considered stale, and both the
    directory and lock file are removed.
  - If TryLock() fails: The directory is considered in active use by another
    process/goroutine, and it's skipped.
  - If lock file is not found, the directory is considered stale and removed.
  - If directory is not found, the lock file is considered stale and removed.

3. TempDir Usage:
  - TempDir.Add() requires the instance lock to be held (enforced by AssertLocked).
  - If the instance's temporary directory hasn't been initialized yet, Add will
    call initInstanceTempDir to create and lock it.
  - Files moved into the temporary directory are renamed with a timestamp prefix
    to ensure uniqueness.
  - If TempDir was cleaned up, Add() will reinitialize the new instance lock and
    new temporary directory.

4. Cleanup Process:
  - TempDir.Cleanup() verifies the instance lock is held (with AssertLocked).
  - It removes both the temporary directory and its lock file.
  - The instance lock is unlocked after cleanup operations are complete.
  - The TempDir instance becomes inactive after cleanup (internal fields are reset).
  - The Add() method will reinitialize the instance lock and temporary directory
    if called after Cleanup().

5. TempDir Lifetime:
  - NewTempDir() only creates a TempDir manager but doesn't actually create the
    instance-specific temporary directory. It performs stale directory recovery.
  - The actual temporary directory is created lazily on the first call to Add().
  - During its lifetime, the temporary directory is protected by its instance lock.
  - The temporary directory exists until Cleanup() is called, which removes both
    the directory and its lock file.
  - Multiple TempDir instances can coexist in the same RootDir, each with its own
    unique subdirectory and lock.
  - After cleanup, the TempDir's temp directory and lock file can't be reused (They are removed) - a new one is created by Add()
    and cleanup must be performed again.

6. Example Directory Structure:

	RootDir/
	    lock-ABC           (instance lock for temp-dir-ABC)
	    temp-dir-ABC/
	        file1
	        file3
	    lock-XYZ           (instance lock for temp-dir-XYZ)
	    temp-dir-XYZ/
	        file2
*/
const (
	// TempDirPrefix is the prefix used for creating temporary directories.
	TempDirPrefix = "temp-dir-"
	// TempdirLockPrefix is the prefix used for creating lock files for temporary directories.
	TempdirLockPrefix = "lock-"
)

// TempDir represents a temporary directory that is created in a specified root directory.
// It manages the lifecycle of the temporary directory, including creation, locking, and cleanup.
// Each TempDir instance is associated with a unique subdirectory in the root directory.
type TempDir struct {
	RootDir string

	tempDirPath string
	// tempDirLock is a lock file (e.g., RootDir/lock-XYZ) specific to this
	// TempDir instance, indicating it's in active use.
	tempDirLock     *lockfile.LockFile
	tempDirLockPath string

	// Required tempDirLock to be held when accessing counter.
	counter uint64
}

// listPotentialStaleDirs scans the RootDir for directories that might be stale temporary directories.
// It identifies directories with the TempDirPrefix and their corresponding lock files with the TempdirLockPrefix.
// If a directory does not have a corresponding lock file, it is considered stale.
// The function returns a list of IDs, that identifies lock and dir.
func listPotentialStaleDirs(rootDir string) (map[string]struct{}, error) {
	ids := make(map[string]struct{})

	dirContent, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("error reading temp dir %s: %w", rootDir, err)
	}

	for _, entry := range dirContent {
		if id, ok := strings.CutPrefix(entry.Name(), TempDirPrefix); ok {
			ids[id] = struct{}{}
			continue
		}

		if id, ok := strings.CutPrefix(entry.Name(), TempdirLockPrefix); ok {
			ids[id] = struct{}{}
		}
	}
	return ids, err
}

// RecoverStaleDirs identifies and removes stale temporary directories in the root directory.
// A directory is considered stale if its lock file can be acquired (indicating no active use).
// The function attempts to remove both the directory and its lock file.
// Any errors encountered during the recovery process are collected and returned as a single error.
func RecoverStaleDirs(rootDir string) error {
	potentialStaleDirs, err := listPotentialStaleDirs(rootDir)
	if err != nil {
		return fmt.Errorf("error listing potential stale temp dirs in %s: %w", rootDir, err)
	}

	if len(potentialStaleDirs) == 0 {
		return nil
	}

	var recoveryErrors []error

	for id := range potentialStaleDirs {
		lockPath := filepath.Join(rootDir, TempdirLockPrefix+id)
		tempDirPath := filepath.Join(rootDir, TempDirPrefix+id)

		// ---- This is racy: Needs Atomic way to GetLockFile and lock it.
		// Lockfile is created in case file not exists.
		instanceLock, err := lockfile.GetLockFile(lockPath)
		if err != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("error getting lock file %s: %w", lockPath, err))
		}
		if err := instanceLock.TryLock(); err != nil {
			logrus.Debugf("Skipping cleanup for locked directory corresponding to %s", lockPath)
			continue
		}
		// ----

		if rmErr := os.RemoveAll(tempDirPath); rmErr != nil && !os.IsNotExist(rmErr) {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("error removing stale temp dir %s: %w", tempDirPath, rmErr))
		}

		// --- This is racy: Needs atomic unlock and remove.
		instanceLock.Unlock()
		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("error removing stale instance lock file %s: %w", lockPath, err))
		}
		// ---

	}

	return errors.Join(recoveryErrors...)
}

// initInstanceTempDir creates and locks a new unique temporary directory for this TempDir instance.
// It acquires the tempDirGlobalLock during its operation to ensure atomicity
// with other temp dir management operations.
// The caller should ensure .Cleanup() is called eventually.
func (td *TempDir) initInstanceTempDir() error {
	if td.tempDirPath != "" && td.tempDirLock != nil {
		logrus.Debugf("Temp dir %s already initialized", td.tempDirPath)
		return nil
	}

	// ---- This is racy: Needs Atomic way to create uncinate file and lock it.
	// Reserve name for lock file in the RootDir.
	tempLockFile, err := os.CreateTemp(td.RootDir, TempdirLockPrefix)
	if err != nil {
		return fmt.Errorf("creating temp directory in %s failed: %w", td.RootDir, err)
	}
	tempLockFile.Close()

	tempDirLockPath := filepath.Join(td.RootDir, tempLockFile.Name())
	tempDirLock, err := lockfile.GetLockFile(tempDirLockPath)
	if err != nil {
		return fmt.Errorf("getting instance lock file %s for temp dir: %w", tempDirLockPath, err)
	}

	if err := tempDirLock.TryLock(); err != nil {
		return fmt.Errorf("locking instance lock for tempdir failed: %w", err)
	}
	// ----

	id := strings.TrimPrefix(tempLockFile.Name(), TempdirLockPrefix)

	actualTempDirPath := filepath.Join(td.RootDir, TempDirPrefix+id)
	if err := os.MkdirAll(actualTempDirPath, 0o700); err != nil {
		return fmt.Errorf("creating temp directory %s failed: %w", actualTempDirPath, err)
	}

	td.tempDirPath = actualTempDirPath
	td.tempDirLock = tempDirLock
	td.tempDirLockPath = tempDirLockPath
	td.counter = 0
	return nil
}

// NewTempDir prepares a TempDir manager for a temporary directory in the specified RootDir.
// The RootDir itself will be created if it doesn't exist.
// It performs recovery of stale temporary directories within RootDir (removing those not in active use).
// The actual instance-specific temporary directory is created and locked on the first call to Add().
// Stale directory recovery (in NewTempDir) and instance directory creation (in Add via initInstanceTempDir).
func NewTempDir(rootDir string) (*TempDir, error) {
	if err := os.MkdirAll(rootDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating root temp directory %s failed: %w", rootDir, err)
	}

	td := &TempDir{
		RootDir: rootDir,
	}

	if err := RecoverStaleDirs(rootDir); err != nil {
		logrus.Warnf("Error during stale temp dir recovery in %s: %v", rootDir, err)
	}

	return td, nil
}

// Add moves the specified file into the instance's temporary directory.
// If the instance's temporary directory doesn't exist yet, Add will create and lock it.
// Note: 'path' must be on the same filesystem as the TempDir for os.Rename to work.
// The caller MUST ensure .Cleanup() is called.
func (td *TempDir) Add(path string) error {
	if td.tempDirLock == nil {
		if err := td.initInstanceTempDir(); err != nil {
			return fmt.Errorf("initializing instance temp dir failed: %w", err)
		}
	}
	td.tempDirLock.AssertLocked()

	fileName := fmt.Sprintf("%d-", td.counter) + filepath.Base(path)
	destPath := filepath.Join(td.tempDirPath, fileName)
	td.counter++
	return os.Rename(path, destPath)
}

// Cleanup removes the temporary directory and releases its instance lock.
// It verifies the instance lock is held (with AssertLocked), removes the temp directory
// and its lock file, then resets the TempDir instance fields.
// If the temporary directory has already been cleaned up, it logs a debug message and returns nil.
// Callers should typically defer Cleanup() to run after any application-level
// global locks are released to avoid holding those locks during potentially
// slow disk I/O.
func (td *TempDir) Cleanup() error {
	if td.tempDirLock == nil {
		logrus.Debug("Temp dir already cleaned up")
		return nil
	}
	td.tempDirLock.AssertLocked()

	if err := os.RemoveAll(td.tempDirPath); err != nil {
		return fmt.Errorf("removing temp dir %s failed: %w", td.tempDirPath, err)
	}

	// --- This is racy: Needs atomic unlock and remove.
	t := td.tempDirLock
	defer t.Unlock()
	if err := os.Remove(td.tempDirLockPath); err != nil {
		return fmt.Errorf("removing instance lock file %s failed: %w", td.tempDirLockPath, err)
	}
	// ---
	td.tempDirPath = ""
	td.tempDirLock = nil
	td.tempDirLockPath = ""
	return nil
}
