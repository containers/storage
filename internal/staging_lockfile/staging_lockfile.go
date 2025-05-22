package staging_lockfile

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/containers/storage/internal/rawfilelock"
)

// StagingLockFile represents a file lock used to coordinate access to staging areas.
// Typical usage is via CreateAndLock (for a new file) or TryLockExisting (for an existing file),
// both of which return a StagingLockFile that must eventually be released with UnlockAndDelete.
// This ensures that access to the staging file is properly synchronized both within and across processes.
//
// WARNING: This struct MUST NOT be created manually. Use the provided helper functions instead.
type StagingLockFile struct {
	// The following fields are only set when constructing *StagingLockFile, and must never be modified afterwards.
	// They are safe to access without any other locking.
	file string

	// rwMutex serializes concurrent reader-writer acquisitions in the same process space
	rwMutex *sync.RWMutex
	// stateMutex is used to synchronize concurrent accesses to the state below
	stateMutex *sync.Mutex
	locked     bool
	fd         rawfilelock.FileHandle
}

var (
	stagingLockFiles    map[string]*StagingLockFile
	stagingLockFileLock sync.Mutex
)

// AssertLocked checks if the lock is currently held and panics if it's not.
func (l *StagingLockFile) AssertLocked() {
	// DO NOT provide a variant that returns the value of l.locked.
	//
	// If the caller does not hold the lock, l.locked might nevertheless be true because another goroutine does hold it, and
	// we can’t tell the difference.
	//
	// Hence, this “AssertLocked” method, which exists only for sanity checks.

	// Don’t even bother with l.stateMutex: The caller is expected to hold the lock, and in that case l.locked is constant true
	// with no possible writers.
	// If the caller does not hold the lock, we are violating the locking/memory model anyway, and accessing the data
	// without the lock is more efficient for callers, and potentially more visible to lock analysers for incorrect callers.
	if !l.locked {
		panic("internal error: lock is not held by the expected owner")
	}
}

// getLockfile returns a StagingLockFile object associated with the specified path.
// It ensures only one StagingLockFile object exists per path within the process.
// If a StagingLockFile for the path already exists, it returns that instance.
// Otherwise, it creates a new one.
func getLockfile(path string) (*StagingLockFile, error) {
	stagingLockFileLock.Lock()
	defer stagingLockFileLock.Unlock()
	if stagingLockFiles == nil {
		stagingLockFiles = make(map[string]*StagingLockFile)
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("ensuring that path %q is an absolute path: %w", path, err)
	}
	if lockFile, ok := stagingLockFiles[cleanPath]; ok {
		return lockFile, nil
	}
	lockFile, err := createStagingLockFileForPath(cleanPath) // platform-dependent LockFile
	if err != nil {
		return nil, err
	}
	stagingLockFiles[cleanPath] = lockFile
	return lockFile, nil
}

// createStagingLockFileForPath creates a new StagingLockFile instance for the given path.
// It verifies that the file can be opened before returning the StagingLockFile.
// This function will be called at most once for each unique path within a process.
func createStagingLockFileForPath(path string) (*StagingLockFile, error) {
	// Check if we can open the lock.
	fd, err := rawfilelock.OpenLock(path, false)
	if err != nil {
		return nil, err
	}
	rawfilelock.UnlockAndCloseHandle(fd)

	return &StagingLockFile{
		file:       path,
		rwMutex:    &sync.RWMutex{},
		stateMutex: &sync.Mutex{},
		locked:     false,
	}, nil
}

// tryLock attempts to acquire an exclusive lock on the StagingLockFile without blocking.
// It first tries to acquire the internal rwMutex, then opens and tries to lock the file.
// Returns nil on success or an error if any step fails.
func (l *StagingLockFile) tryLock() error {
	success := l.rwMutex.TryLock()
	rwMutexUnlocker := l.rwMutex.Unlock

	if !success {
		return fmt.Errorf("resource temporarily unavailable")
	}
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()
	fd, err := rawfilelock.OpenLock(l.file, false)
	if err != nil {
		rwMutexUnlocker()
		return err
	}
	l.fd = fd

	if err = rawfilelock.TryLockFile(l.fd, rawfilelock.WriteLock); err != nil {
		rawfilelock.CloseHandle(fd)
		rwMutexUnlocker()
		return err
	}

	l.locked = true
	return nil
}
