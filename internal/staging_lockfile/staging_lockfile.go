package staging_lockfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/containers/storage/internal/rawfilelock"
)

// StagingLockFile represents a file lock used to coordinate access to staging areas.
// Typical usage is via CreateAndLock or TryLockPath, both of which return a StagingLockFile
// that must eventually be released with UnlockAndDelete. This ensures that access
// to the staging file is properly synchronized both within and across processes.
//
// WARNING: This struct MUST NOT be created manually. Use the provided helper functions instead.
type StagingLockFile struct {
	// Locking invariant: If stagingLockFileLock is not locked, a StagingLockFile for a particular
	// path exists if the current process currently owns the lock for that file, and it is recorded in stagingLockFiles.
	//
	// The following fields are only set when constructing *StagingLockFile, and must never be modified afterwards.
	// They are safe to access without any other locking.
	file string // Also the key in stagingLockFiles
	fd   rawfilelock.FileHandle
}

const maxRetries = 1000

var (
	stagingLockFiles    map[string]*StagingLockFile
	stagingLockFileLock sync.Mutex
)

// tryAcquireLockForFile attempts to acquire a lock for the specified file path.
func tryAcquireLockForFile(path string) (*StagingLockFile, error) {
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("ensuring that path %q is an absolute path: %w", path, err)
	}

	stagingLockFileLock.Lock()
	defer stagingLockFileLock.Unlock()

	if stagingLockFiles == nil {
		stagingLockFiles = make(map[string]*StagingLockFile)
	}

	if _, ok := stagingLockFiles[cleanPath]; ok {
		return nil, fmt.Errorf("lock %q is used already with other thread", cleanPath)
	}

	fd, err := rawfilelock.OpenLock(cleanPath, false)
	if err != nil {
		return nil, err
	}

	if err = rawfilelock.TryLockFile(fd, rawfilelock.WriteLock); err != nil {
		// Lock acquisition failed, but holding stagingLockFileLock ensures
		// no other goroutine in this process is using this file descriptor,
		// so closing it is still safe.
		rawfilelock.CloseHandle(fd)
		return nil, fmt.Errorf("failed to acquire lock on %q: %w", cleanPath, err)
	}

	lockFile := &StagingLockFile{
		file: cleanPath,
		fd:   fd,
	}

	stagingLockFiles[cleanPath] = lockFile
	return lockFile, nil
}

// UnlockAndDelete releases the lock, removes the associated file from the filesystem.
//
// WARNING: After this operation, the StagingLockFile becomes invalid for further use.
func (l *StagingLockFile) UnlockAndDelete() error {
	stagingLockFileLock.Lock()
	defer stagingLockFileLock.Unlock()

	if l.file == "" {
		// Panic when unlocking an unlocked lock. That's a violation
		// of the lock semantics and will reveal such.
		panic("calling Unlock on unlocked lock")
	}

	defer func() {
		// It’s important that this happens while we are still holding stagingLockFileLock, to ensure
		// that no other goroutine has l.file open = that this close is not unlocking the lock under any
		// other goroutine. (defer ordering is LIFO, so this will happen before we release the stagingLockFileLock)
		rawfilelock.UnlockAndCloseHandle(l.fd)
		delete(stagingLockFiles, l.file)
		l.file = ""
	}()
	if err := os.Remove(l.file); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CreateAndLock creates a new temporary file in the specified directory with the given pattern,
// then creates and locks a StagingLockFile for it. The file is created using os.CreateTemp.
// Typically, the caller would use the returned lock file path to derive a path to the lock-controlled resource
// (e.g. by replacing the "pattern" part of the returned file name with a different prefix)
// Caller MUST call UnlockAndDelete() on the returned StagingLockFile to release the lock and delete the file.
//
// Returns:
//   - The locked StagingLockFile
//   - The path to the created file as presented to Open.
//   - Any error that occurred during the process
//
// If the file cannot be locked, this function will retry up to maxRetries times before failing.
func CreateAndLock(dir string, pattern string) (*StagingLockFile, string, error) {
	for try := 0; ; try++ {
		file, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, "", err
		}
		file.Close()

		l, err := tryAcquireLockForFile(file.Name())
		if err != nil {
			if try < maxRetries {
				continue // Retry if the lock cannot be acquired
			}
			return nil, "", fmt.Errorf(
				"failed to allocate lock in %q after %d attempts; last failure on %q: %w",
				dir, try, l.file, err,
			)
		}

		return l, file.Name(), nil
	}
}

// TryLockPath attempts to acquire a lock on an specific path. If the file does not exist,
// it will be created.
//
// Warning: If acquiring a lock is successful, it returns a new StagingLockFile
// instance for the file. Caller MUST call UnlockAndDelete() on the returned StagingLockFile
// to release the lock and delete the file.
func TryLockPath(path string) (*StagingLockFile, error) {
	return tryAcquireLockForFile(path)
}
