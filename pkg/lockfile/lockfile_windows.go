//go:build windows
// +build windows

package lockfile

import (
	"os"
	"sync"
	"time"
)

// createLockFileForPath returns a *LockFile object, possibly (depending on the platform)
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
	return &LockFile{locked: false}, nil
}

// *LockFile represents a file lock where the file is used to cache an
// identifier of the last party that made changes to whatever's being protected
// by the lock.
//
// It MUST NOT be created manually. Use GetLockFile or GetROLockFile instead.
type LockFile struct {
	mu     sync.Mutex
	file   string
	locked bool
}

func (l *LockFile) Lock() {
	l.mu.Lock()
	l.locked = true
}

func (l *LockFile) RLock() {
	l.mu.Lock()
	l.locked = true
}

func (l *LockFile) Unlock() {
	l.locked = false
	l.mu.Unlock()
}

func (l *LockFile) AssertLocked() {
	// DO NOT provide a variant that returns the value of l.locked.
	//
	// If the caller does not hold the lock, l.locked might nevertheless be true because another goroutine does hold it, and
	// we can’t tell the difference.
	//
	// Hence, this “AssertLocked” method, which exists only for sanity checks.
	if !l.locked {
		panic("internal error: lock is not held by the expected owner")
	}
}

func (l *LockFile) AssertLockedForWriting() {
	// DO NOT provide a variant that returns the current lock state.
	//
	// The same caveats as for AssertLocked apply equally.
	l.AssertLocked() // The current implementation does not distinguish between read and write locks.
}

func (l *LockFile) Modified() (bool, error) {
	return false, nil
}
func (l *LockFile) Touch() error {
	return nil
}
func (l *LockFile) IsReadWrite() bool {
	return false
}

func (l *LockFile) TouchedSince(when time.Time) bool {
	stat, err := os.Stat(l.file)
	if err != nil {
		return true
	}
	return when.Before(stat.ModTime())
}
