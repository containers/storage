//go:build windows

package filelock

import (
	"golang.org/x/sys/windows"
)

const (
	reserved = 0
	allBytes = ^uint32(0)
)

type FileHandle windows.Handle

func openHandle(path string, mode int) (FileHandle, error) {
	mode |= windows.O_CLOEXEC
	fd, err := windows.Open(path, mode, windows.S_IWRITE)
	return FileHandle(fd), err
}

func lockHandle(fd FileHandle, lType LockType, nonblocking bool) error {
	flags := 0
	if lType != ReadLock {
		flags = windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	if nonblocking {
		flags |= windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	ol := new(windows.Overlapped)
	if err := windows.LockFileEx(windows.Handle(fd), uint32(flags), reserved, allBytes, allBytes, ol); err != nil {
		if nonblocking {
			return err
		}
		panic(err)
	}
	return nil
}

func unlockAndCloseHandle(fd FileHandle) {
	ol := new(windows.Overlapped)
	windows.UnlockFileEx(windows.Handle(fd), reserved, allBytes, allBytes, ol)
	closeHandle(fd)
}

func closeHandle(fd FileHandle) {
	windows.Close(windows.Handle(fd))
}
