package staging_lockfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type lockType byte

const (
	readLock lockType = iota
	writeLock
)

// LockFile represents a file lock where the file is used to cache an
// identifier of the last party that made changes to whatever's being protected
// by the lock.
//
// It MUST NOT be created manually. Use GetLockFile or GetROLockFile instead.
type LockFile struct {
	// The following fields are only set when constructing *LockFile, and must never be modified afterwards.
	// They are safe to access without any other locking.
	file string
	ro   bool

	// rwMutex serializes concurrent reader-writer acquisitions in the same process space
	rwMutex *sync.RWMutex
	// stateMutex is used to synchronize concurrent accesses to the state below
	stateMutex *sync.Mutex
	counter    int64
	//lw         LastWrite // A global value valid as of the last .Touch() or .Modified()
	lockType lockType
	locked   bool
	// The following fields are only modified on transitions between counter == 0 / counter != 0.
	// Thus, they can be safely accessed by users _that currently hold the LockFile_ without locking.
	// In other cases, they need to be protected using stateMutex.
	fd fileHandle
}

var (
	lockFiles     map[string]*LockFile
	lockFilesLock sync.Mutex
)

// GetLockFile opens a read-write lock file, creating it if necessary.  The
// *LockFile object may already be locked if the path has already been requested
// by the current process.
func GetLockFile(path string) (*LockFile, error) {
	return getLockfile(path, false)
}

// GetROLockFile opens a read-only lock file, creating it if necessary.  The
// *LockFile object may already be locked if the path has already been requested
// by the current process.
func GetROLockFile(path string) (*LockFile, error) {
	return getLockfile(path, true)
}

// Lock locks the lockfile as a writer.  Panic if the lock is a read-only one.
func (l *LockFile) Lock() {
	if l.ro {
		panic("can't take write lock on read-only lock file")
	}
	l.lock(writeLock)
}

// RLock locks the lockfile as a reader.
func (l *LockFile) RLock() {
	l.lock(readLock)
}

// TryLock attempts to lock the lockfile as a writer.  Panic if the lock is a read-only one.
func (l *LockFile) TryLock() error {
	if l.ro {
		panic("can't take write lock on read-only lock file")
	}
	return l.tryLock(writeLock)
}

// TryRLock attempts to lock the lockfile as a reader.
func (l *LockFile) TryRLock() error {
	return l.tryLock(readLock)
}

// Unlock unlocks the lockfile.
func (l *LockFile) Unlock() {
	l.stateMutex.Lock()
	if !l.locked {
		// Panic when unlocking an unlocked lock.  That's a violation
		// of the lock semantics and will reveal such.
		panic("calling Unlock on unlocked lock")
	}
	l.counter--
	if l.counter < 0 {
		// Panic when the counter is negative.  There is no way we can
		// recover from a corrupted lock and we need to protect the
		// storage from corruption.
		panic(fmt.Sprintf("lock %q has been unlocked too often", l.file))
	}
	if l.counter == 0 {
		// We should only release the lock when the counter is 0 to
		// avoid releasing read-locks too early; a given process may
		// acquire a read lock multiple times.
		l.locked = false
		// Close the file descriptor on the last unlock, releasing the
		// file lock.
		unlockAndCloseHandle(l.fd)
	}
	if l.lockType == readLock {
		l.rwMutex.RUnlock()
	} else {
		l.rwMutex.Unlock()
	}
	l.stateMutex.Unlock()
}

func (l *LockFile) AssertLocked() {
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

func (l *LockFile) AssertLockedForWriting() {
	// DO NOT provide a variant that returns the current lock state.
	//
	// The same caveats as for AssertLocked apply equally.

	l.AssertLocked()
	// Like AssertLocked, don’t even bother with l.stateMutex.
	if l.lockType == readLock {
		panic("internal error: lock is not held for writing")
	}
}

// IsReadWrite indicates if the lock file is a read-write lock.
func (l *LockFile) IsReadWrite() bool {
	return !l.ro
}

// getLockFile returns a *LockFile object, possibly (depending on the platform)
// working inter-process, and associated with the specified path.
//
// If ro, the lock is a read-write lock and the returned *LockFile should correspond to the
// “lock for reading” (shared) operation; otherwise, the lock is either an exclusive lock,
// or a read-write lock and *LockFile should correspond to the “lock for writing” (exclusive) operation.
//
// WARNING:
// - The lock may or MAY NOT be inter-process.
// - There may or MAY NOT be an actual object on the filesystem created for the specified path.
// - Even if ro, the lock MAY be exclusive.
func getLockfile(path string, ro bool) (*LockFile, error) {
	lockFilesLock.Lock()
	defer lockFilesLock.Unlock()
	if lockFiles == nil {
		lockFiles = make(map[string]*LockFile)
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("ensuring that path %q is an absolute path: %w", path, err)
	}
	if lockFile, ok := lockFiles[cleanPath]; ok {
		if ro && lockFile.IsReadWrite() {
			return nil, fmt.Errorf("lock %q is not a read-only lock", cleanPath)
		}
		if !ro && !lockFile.IsReadWrite() {
			return nil, fmt.Errorf("lock %q is not a read-write lock", cleanPath)
		}
		return lockFile, nil
	}
	lockFile, err := createLockFileForPath(cleanPath, ro) // platform-dependent LockFile
	if err != nil {
		return nil, err
	}
	lockFiles[cleanPath] = lockFile
	return lockFile, nil
}

// createLockFileForPath returns new *LockFile object, possibly (depending on the platform)
// working inter-process and associated with the specified path.
//
// This function will be called at most once for each path value within a single process.
//
// If ro, the lock is a read-write lock and the returned *LockFile should correspond to the
// “lock for reading” (shared) operation; otherwise, the lock is either an exclusive lock,
// or a read-write lock and *LockFile should correspond to the “lock for writing” (exclusive) operation.
//
// WARNING:
// - The lock may or MAY NOT be inter-process.
// - There may or MAY NOT be an actual object on the filesystem created for the specified path.
// - Even if ro, the lock MAY be exclusive.
func createLockFileForPath(path string, ro bool) (*LockFile, error) {
	// Check if we can open the lock.
	fd, err := openLock(path, ro)
	if err != nil {
		return nil, err
	}
	unlockAndCloseHandle(fd)

	lType := writeLock
	if ro {
		lType = readLock
	}

	return &LockFile{
		file: path,
		ro:   ro,

		rwMutex:    &sync.RWMutex{},
		stateMutex: &sync.Mutex{},
		//lw:         newLastWrite(), // For compatibility, the first call of .Modified() will always report a change.
		lockType: lType,
		locked:   false,
	}, nil
}

// openLock opens the file at path and returns the corresponding file
// descriptor. The path is opened either read-only or read-write,
// depending on the value of ro argument.
//
// openLock will create the file and its parent directories,
// if necessary.
func openLock(path string, ro bool) (fd fileHandle, err error) {
	flags := os.O_CREATE
	if ro {
		flags |= os.O_RDONLY
	} else {
		flags |= os.O_RDWR
	}
	fd, err = openHandle(path, flags)
	if err == nil {
		return fd, nil
	}

	// the directory of the lockfile seems to be removed, try to create it
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return fd, fmt.Errorf("creating lock file directory: %w", err)
		}

		return openLock(path, ro)
	}

	return fd, &os.PathError{Op: "open", Path: path, Err: err}
}

// lock locks the lockfile via syscall based on the specified type and
// command.
func (l *LockFile) lock(lType lockType) {
	if lType == readLock {
		l.rwMutex.RLock()
	} else {
		l.rwMutex.Lock()
	}
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()
	if l.counter == 0 {
		// If we're the first reference on the lock, we need to open the file again.
		fd, err := openLock(l.file, l.ro)
		if err != nil {
			panic(err)
		}
		l.fd = fd

		// Optimization: only use the (expensive) syscall when
		// the counter is 0.  In this case, we're either the first
		// reader lock or a writer lock.
		if err := lockHandle(l.fd, lType, false); err != nil {
			panic(err)
		}
	}
	l.lockType = lType
	l.locked = true
	l.counter++
}

// lock locks the lockfile via syscall based on the specified type and
// command.
func (l *LockFile) tryLock(lType lockType) error {
	var success bool
	var rwMutexUnlocker func()
	if lType == readLock {
		success = l.rwMutex.TryRLock()
		rwMutexUnlocker = l.rwMutex.RUnlock
	} else {
		success = l.rwMutex.TryLock()
		rwMutexUnlocker = l.rwMutex.Unlock
	}
	if !success {
		return fmt.Errorf("resource temporarily unavailable")
	}
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()
	if l.counter == 0 {
		// If we're the first reference on the lock, we need to open the file again.
		fd, err := openLock(l.file, l.ro)
		if err != nil {
			rwMutexUnlocker()
			return err
		}
		l.fd = fd

		// Optimization: only use the (expensive) syscall when
		// the counter is 0.  In this case, we're either the first
		// reader lock or a writer lock.
		if err = lockHandle(l.fd, lType, true); err != nil {
			closeHandle(fd)
			rwMutexUnlocker()
			return err
		}
	}
	l.lockType = lType
	l.locked = true
	l.counter++
	return nil
}
