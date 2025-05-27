package filelock

import (
	"fmt"
	"os"
	"path/filepath"
)

type LockType byte

const (
	ReadLock LockType = iota
	WriteLock
)

// OpenLock opens a file for locking, creating parent directories if needed
func OpenLock(path string, readOnly bool) (FileHandle, error) {
	flags := os.O_CREATE
	if readOnly {
		flags |= os.O_RDONLY
	} else {
		flags |= os.O_RDWR
	}

	fd, err := openHandle(path, flags)
	if err == nil {
		return fd, nil
	}

	// the directory of the lockfile seems to be removed, try to create it
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return fd, fmt.Errorf("creating lock file directory: %w", err)
		}

		return OpenLock(path, readOnly)
	}

	return fd, &os.PathError{Op: "open", Path: path, Err: err}
}

// TryLockFile attempts to lock a file handle
func TryLockFile(fd FileHandle, lockType LockType) error {
	return lockHandle(fd, lockType, true)
}

// LockFile locks a file handle
func LockFile(fd FileHandle, lockType LockType) error {
	return lockHandle(fd, lockType, false)
}

// UnlockAndClose unlocks and closes a file handle
func UnlockAndCloseHandle(fd FileHandle) {
	unlockAndCloseHandle(fd)
}

// CloseHandle closes a file handle without unlocking
func CloseHandle(fd FileHandle) {
	closeHandle(fd)
}
