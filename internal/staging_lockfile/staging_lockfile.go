package staging_lockfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/containers/storage/internal/filelock"
)

// StagingLockFile represents a file lock used to coordinate access to staging areas.
// It implements both in-process locking (via rwMutex) and inter-process locking
// using filesystem locks.
//
// It MUST NOT be created manually. Use GetStagingLockFile instead.
type StagingLockFile struct {
	// The following fields are only set when constructing *StagingLockFile, and must never be modified afterwards.
	// They are safe to access without any other locking.
	file string

	// rwMutex serializes concurrent reader-writer acquisitions in the same process space
	rwMutex *sync.RWMutex
	// stateMutex is used to synchronize concurrent accesses to the state below
	stateMutex *sync.Mutex
	locked     bool
	fd         filelock.FileHandle
}

const maxRetries = 1000

var (
	stagingLockFile     map[string]*StagingLockFile
	stagingLockFileLock sync.Mutex
)

// GetStagingLockFile returns a StagingLockFile for the specified path, creating it if necessary.
// If the path has already been requested by the current process, the same StagingLockFile instance
// is returned, which may already be locked by another goroutine.
// The lock file on disk is created if it doesn't exist, along with any necessary parent directories.
func GetStagingLockFile(path string) (*StagingLockFile, error) {
	return getLockfile(path)
}

// Lock acquires an exclusive write lock on the file.
// It blocks until the lock can be acquired.
// If the lock cannot be acquired due to file system errors, it will panic.
func (l *StagingLockFile) Lock() {
	l.lock()
}

// TryLock attempts to lock the file without blocking.
// It returns nil if the lock was successfully acquired, or an error if the lock
// is held by another process or goroutine or if a file system error occurs.
func (l *StagingLockFile) TryLock() error {
	return l.tryLock()
}

// Unlock releases the lock on the file.
// It will panic if the lock is not currently held by the caller.
func (l *StagingLockFile) Unlock() {
	l.stateMutex.Lock()
	if !l.locked {
		// Panic when unlocking an unlocked lock.  That's a violation
		// of the lock semantics and will reveal such.
		panic("calling Unlock on unlocked lock")
	}
	l.locked = false
	// Close the file descriptor on unlock, releasing the file lock
	filelock.UnlockAndCloseHandle(l.fd)
	l.rwMutex.Unlock()
	l.stateMutex.Unlock()
}

// AssertLocked checks if the lock is currently held and panics if it's not.
// This is intended for sanity checks in code that requires the lock to be held.
// Note: This method does not acquire stateMutex as it assumes the caller
// already holds the lock, in which case l.locked is guaranteed to be true.
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
//
// WARNING:
// - The lock may or MAY NOT be inter-process.
// - There may or MAY NOT be an actual object on the filesystem created for the specified path.
func getLockfile(path string) (*StagingLockFile, error) {
	stagingLockFileLock.Lock()
	defer stagingLockFileLock.Unlock()
	if stagingLockFile == nil {
		stagingLockFile = make(map[string]*StagingLockFile)
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("ensuring that path %q is an absolute path: %w", path, err)
	}
	if lockFile, ok := stagingLockFile[cleanPath]; ok {
		return lockFile, nil
	}
	lockFile, err := createStagingLockFileForPath(cleanPath) // platform-dependent LockFile
	if err != nil {
		return nil, err
	}
	stagingLockFile[cleanPath] = lockFile
	return lockFile, nil
}

// createStagingLockFileForPath creates a new StagingLockFile instance for the given path.
// It verifies that the file can be opened before returning the StagingLockFile.
// This function will be called at most once for each unique path within a process.
//
// WARNING:
// - The lock may or MAY NOT be inter-process.
// - There may or MAY NOT be an actual object on the filesystem created for the specified path.
func createStagingLockFileForPath(path string) (*StagingLockFile, error) {
	// Check if we can open the lock.
	fd, err := filelock.OpenLock(path, false)
	if err != nil {
		return nil, err
	}
	filelock.UnlockAndCloseHandle(fd)

	return &StagingLockFile{
		file:       path,
		rwMutex:    &sync.RWMutex{},
		stateMutex: &sync.Mutex{},
		locked:     false,
	}, nil
}

// lock acquires an exclusive lock on the StagingLockFile.
// It first acquires the internal rwMutex, then opens and locks the file.
// If any step fails, it will panic.
func (l *StagingLockFile) lock() {
	l.rwMutex.Lock()
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()
	fd, err := filelock.OpenLock(l.file, false)
	if err != nil {
		panic(err)
	}
	l.fd = fd

	if err := filelock.LockFile(l.fd, filelock.WriteLock); err != nil {
		panic(err)
	}
	l.locked = true
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
	fd, err := filelock.OpenLock(l.file, false)
	if err != nil {
		rwMutexUnlocker()
		return err
	}
	l.fd = fd

	if err = filelock.TryLockFile(l.fd, filelock.WriteLock); err != nil {
		filelock.CloseHandle(fd)
		rwMutexUnlocker()
		return err
	}

	l.locked = true
	return nil
}

// UnlockAndDelete releases the lock, removes the associated file from the filesystem,
// and removes this StagingLockFile from the global cache.
//
// Panics if:
//   - The lock is not currently held by the caller.
//   - The file cannot be removed (and it exists).
//
// WARNING: After this operation, the StagingLockFile becomes invalid for further use as the file field is cleared.
// A new call to GetStagingLockFile with the same path will create a new instance.
func (l *StagingLockFile) UnlockAndDelete() {
	l.stateMutex.Lock()
	if !l.locked {
		// Panic when unlocking an unlocked lock.  That's a violation
		// of the lock semantics and will reveal such.
		panic("calling Unlock on unlocked lock")
	}

	l.locked = false

	if err := os.Remove(l.file); err != nil && !os.IsNotExist(err) {
		panic(fmt.Errorf("removing lock file %q: %w", l.file, err))
	}

	path := l.file
	l.file = ""

	stagingLockFileLock.Lock()
	defer stagingLockFileLock.Unlock()
	delete(stagingLockFile, path)

	filelock.UnlockAndCloseHandle(l.fd)
	l.rwMutex.Unlock()
	l.stateMutex.Unlock()
}

// CreateAndLock creates a new temporary file in the specified directory with the given pattern,
// then creates and locks a StagingLockFile for it. The file is created using os.CreateTemp.
// If dir is empty, the system's default temporary directory is used.
//
// Returns:
//   - The locked StagingLockFile
//   - The absolute path to the created file
//   - Any error that occurred during the process
//
// The created file will be registered in the global cache of StagingLockFiles.
// If the file cannot be locked, this function will retry up to maxRetries times before failing.
func CreateAndLock(dir string, pattern string) (*StagingLockFile, string, error) {
	try := 0
	for {
		file, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, "", err
		}
		file.Close()

		cleanPath, err := filepath.Abs(file.Name())
		if err != nil {
			return nil, "", err
		}

		l, err := getLockfile(cleanPath)
		if err != nil {
			panic(err)
		}

		if err := l.TryLock(); err != nil {
			if try++; try < maxRetries {
				continue // Retry if the lock cannot be acquired
			}
			return nil, cleanPath, fmt.Errorf("failed to acquire lock on %q after %d attempts: %w", cleanPath, try, err)
		}

		return l, cleanPath, nil
	}
}

// TryLockExisting attempts to acquire a lock on an existing file without blocking and without using global cache.
// It first checks if the file exists, then get StagingLockFile and tries to lock it.
//
// Returns:
//   - The locked StagingLockFile if successful
//   - An error if the lock cannot be acquired (e.g., if the file is already locked or does not exist).
func TryLockExisting(path string) (*StagingLockFile, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	l, err := createStagingLockFileForPath(path) // platform-dependent LockFile
	if err != nil {
		return nil, err
	}

	if err := l.TryLock(); err != nil {
		return nil, fmt.Errorf("failed to acquire lock on %q: %w", path, err)
	}

	return l, nil
}
