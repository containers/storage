package storage

import (
	"os"
	"sync"
	"syscall"
	"time"
)

type lockfile struct {
	file string
	fd   uintptr
}

func GetLockfile(path string) (sync.Locker, error) {
	if fd, err := syscall.Open(path, os.O_RDWR|os.O_CREATE, syscall.S_IRUSR|syscall.S_IWUSR); err != nil {
		return nil, err
	} else {
		return &lockfile{file: path, fd: uintptr(fd)}, nil
	}
}

func (l *lockfile) Lock() {
	lk := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
	}
	for syscall.FcntlFlock(l.fd, syscall.F_SETLKW, &lk) != nil {
		time.Sleep(10 * time.Millisecond)
	}
}

func (l *lockfile) Unlock() {
	lk := syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
	}
	for syscall.FcntlFlock(l.fd, syscall.F_SETLKW, &lk) != nil {
		time.Sleep(10 * time.Millisecond)
	}
}
