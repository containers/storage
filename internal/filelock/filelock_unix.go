//go:build !windows

package filelock

import (
	"time"

	"golang.org/x/sys/unix"
)

type FileHandle uintptr

func openHandle(path string, mode int) (FileHandle, error) {
	mode |= unix.O_CLOEXEC
	fd, err := unix.Open(path, mode, 0o644)
	return FileHandle(fd), err
}

func lockHandle(fd FileHandle, lType LockType, nonblocking bool) error {
	fType := unix.F_RDLCK
	if lType != ReadLock {
		fType = unix.F_WRLCK
	}
	lk := unix.Flock_t{
		Type:   int16(fType),
		Whence: int16(unix.SEEK_SET),
		Start:  0,
		Len:    0,
	}
	cmd := unix.F_SETLKW
	if nonblocking {
		cmd = unix.F_SETLK
	}
	for {
		err := unix.FcntlFlock(uintptr(fd), cmd, &lk)
		if err == nil || nonblocking {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func unlockAndCloseHandle(fd FileHandle) {
	unix.Close(int(fd))
}

func closeHandle(fd FileHandle) {
	unix.Close(int(fd))
}
