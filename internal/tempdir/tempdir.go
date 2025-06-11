package tempdir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/storage/internal/staging_lockfile"
	"github.com/sirupsen/logrus"
)

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
    attempts to TryLockPath() its instance lock file.
  - If TryLockPath() succeeds: The directory is considered stale, and both the
    directory and lock file are removed.
  - If TryLockPath() fails: The directory is considered in active use by another
    process/goroutine, and it's skipped.

3. TempDir Usage:
  - TempDir.Add() requires the instance lock to be held.
  - If the instance's temporary directory hasn't been initialized yet, Add will
    call initInstanceTempDir to create and lock it.
  - Files moved into the temporary directory are renamed with a counter-based prefix
    to ensure uniqueness (e.g., "0-filename", "1-filename").
  - If TempDir was cleaned up, Add() will reinitialize a new instance lock and
    new temporary directory.

4. Cleanup Process:
  - TempDir.Cleanup() verifies the instance lock is held.
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
  - After cleanup, a new temporary directory with a new ID will be created on the next
    Add() call.

6. Example Directory Structure:

	RootDir/
	    lock-ABC           (instance lock for temp-dir-ABC)
	    temp-dir-ABC/
	        0-file1
	        1-file3
	    lock-XYZ           (instance lock for temp-dir-XYZ)
	    temp-dir-XYZ/
	        0-file2
*/
const (
	// tempDirPrefix is the prefix used for creating temporary directories.
	tempDirPrefix = "temp-dir-"
	// tempdirLockPrefix is the prefix used for creating lock files for temporary directories.
	tempdirLockPrefix = "lock-"
)

// TempDir represents a temporary directory that is created in a specified root directory.
// It manages the lifecycle of the temporary directory, including creation, locking, and cleanup.
// Each TempDir instance is associated with a unique subdirectory in the root directory.
// Warning: The TempDir instance should be used in a single goroutine.
type TempDir struct {
	RootDir string

	tempDirPath string
	// tempDirLock is a lock file (e.g., RootDir/lock-XYZ) specific to this
	// TempDir instance, indicating it's in active use.
	tempDirLock     *staging_lockfile.StagingLockFile
	tempDirLockPath string

	// counter is used to generate unique filenames for added files.
	counter uint64
}

// CleanupTempDirFunc is a function type that can be returned by operations
// which need to perform cleanup actions later.
type CleanupTempDirFunc func() error

// listPotentialStaleDirs scans the RootDir for directories that might be stale temporary directories.
// It identifies directories with the TempDirPrefix and their corresponding lock files with the TempdirLockPrefix.
// The function returns a map of IDs that correspond to both directories and lock files found.
// These IDs are extracted from the filenames by removing their respective prefixes.
func listPotentialStaleDirs(rootDir string) (map[string]struct{}, error) {
	ids := make(map[string]struct{})

	dirContent, err := os.ReadDir(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading temp dir %s: %w", rootDir, err)
	}

	for _, entry := range dirContent {
		if id, ok := strings.CutPrefix(entry.Name(), tempDirPrefix); ok {
			ids[id] = struct{}{}
			continue
		}

		if id, ok := strings.CutPrefix(entry.Name(), tempdirLockPrefix); ok {
			ids[id] = struct{}{}
		}
	}
	return ids, nil
}

// RecoverStaleDirs identifies and removes stale temporary directories in the root directory.
// A directory is considered stale if its lock file can be acquired (indicating no active use).
// The function attempts to remove both the directory and its lock file.
// If a directory's lock cannot be acquired, it is considered in use and is skipped.
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
		lockPath := filepath.Join(rootDir, tempdirLockPrefix+id)
		tempDirPath := filepath.Join(rootDir, tempDirPrefix+id)

		// Try to lock the lock file. If it can be locked, the directory is stale.
		instanceLock, err := staging_lockfile.TryLockPath(lockPath)
		if err != nil {
			continue
		}

		if rmErr := os.RemoveAll(tempDirPath); rmErr != nil && !os.IsNotExist(rmErr) {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("error removing stale temp dir %s: %w", tempDirPath, rmErr))
		}
		if unlockErr := instanceLock.UnlockAndDelete(); unlockErr != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("error unlocking and deleting stale lock file %s: %w", lockPath, unlockErr))
		}
	}

	return errors.Join(recoveryErrors...)
}

// initInstanceTempDir creates and locks a new unique temporary directory for this TempDir instance.
// The caller should ensure .Cleanup() is called eventually.
func (td *TempDir) initInstanceTempDir() error {
	if td.tempDirPath != "" && td.tempDirLock != nil {
		logrus.Debugf("Temp dir %s already initialized", td.tempDirPath)
		return nil
	}

	tempDirLock, tempDirLockFileName, err := staging_lockfile.CreateAndLock(td.RootDir, tempdirLockPrefix)
	if err != nil {
		return fmt.Errorf("creating and locking temp dir instance lock in %s failed: %w", td.RootDir, err)
	}
	td.tempDirLock = tempDirLock
	td.tempDirLockPath = filepath.Join(td.RootDir, tempDirLockFileName)

	// Create the temporary directory that corresponds to the lock file
	id := strings.TrimPrefix(tempDirLockFileName, tempdirLockPrefix)
	actualTempDirPath := filepath.Join(td.RootDir, tempDirPrefix+id)
	if err := os.MkdirAll(actualTempDirPath, 0o700); err != nil {
		return fmt.Errorf("creating temp directory %s failed: %w", actualTempDirPath, err)
	}
	td.tempDirPath = actualTempDirPath
	td.counter = 0
	return nil
}

// NewTempDir prepares a TempDir manager for a temporary directory in the specified RootDir.
// The RootDir itself will be created if it doesn't exist.
// It performs recovery of stale temporary directories within RootDir (removing those not in active use).
// The actual instance-specific temporary directory is created lazily on the first call to Add().
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
// Files are renamed with a counter-based prefix (e.g., "0-filename", "1-filename") to ensure uniqueness.
// Note: 'path' must be on the same filesystem as the TempDir for os.Rename to work.
// The caller MUST ensure .Cleanup() is called.
func (td *TempDir) Add(path string) error {
	if td.tempDirLock == nil {
		if err := td.initInstanceTempDir(); err != nil {
			return fmt.Errorf("initializing instance temp dir failed: %w", err)
		}
	}

	fileName := fmt.Sprintf("%d-", td.counter) + filepath.Base(path)
	destPath := filepath.Join(td.tempDirPath, fileName)
	td.counter++
	return os.Rename(path, destPath)
}

// Cleanup removes the temporary directory and releases its instance lock.
// Callers should typically defer Cleanup() to run after any application-level
// global locks are released to avoid holding those locks during potentially
// slow disk I/O.
func (td *TempDir) Cleanup() error {
	if td.tempDirLock == nil {
		logrus.Debug("Temp dir already cleaned up")
		return nil
	}

	if err := os.RemoveAll(td.tempDirPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing temp dir %s failed: %w", td.tempDirPath, err)
	}

	lock := td.tempDirLock
	td.tempDirPath = ""
	td.tempDirLock = nil
	td.tempDirLockPath = ""
	return lock.UnlockAndDelete()
}

// CleanupTemporaryDirectories cleans up multiple temporary directories by calling their cleanup functions.
func CleanupTemporaryDirectories(cleanFuncs ...CleanupTempDirFunc) error {
	var cleanupErrors []error
	for _, cleanupFunc := range cleanFuncs {
		if cleanupFunc == nil {
			continue
		}
		if err := cleanupFunc(); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}
