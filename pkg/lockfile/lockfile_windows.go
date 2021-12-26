//go:build windows
// +build windows

package lockfile

import (
	"fmt"
	"github.com/containers/storage/pkg/stringid"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	modkernel32    = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx = modkernel32.NewProc("LockFileEx")
)

const (
	LOCKFILE_EXCLUSIVE_LOCK = uint32(2)
	MAXDWORD                = 0xffffffff
)

func openLock(path string, ro bool) (fd syscall.Handle, err error) {
	wstringPath, err := syscall.UTF16FromString(path)
	if err != nil {
		return syscall.Handle(0), err
	}

	mode := uint32(syscall.GENERIC_READ | syscall.GENERIC_WRITE)
	if ro {
		mode = syscall.GENERIC_READ
	}

	fd, err = syscall.CreateFile(&wstringPath[0], mode,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL,
		0)

	if err == nil {
		return
	}

	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return fd, errors.Wrap(err, "creating locker directory")
		}

		return openLock(path, ro)
	}

	return
}

// createLockerForPath returns a Locker object, possibly (depending on the platform)
// working inter-process and associated with the specified path.
//
// This function will be called at most once for each path value within a single process.
//
// If ro, the lock is a read-write lock and the returned Locker should correspond to the
// “lock for reading” (shared) operation; otherwise, the lock is either an exclusive lock,
// or a read-write lock and Locker should correspond to the “lock for writing” (exclusive) operation.
//
// WARNING:
// - The lock may or MAY NOT be inter-process.
// - There may or MAY NOT be an actual object on the filesystem created for the specified path.
// - Even if ro, the lock MAY be exclusive.
func createLockerForPath(path string, ro bool) (Locker, error) {
	// Check if we can open the lock.
	fd, err := openLock(path, ro)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening %q", path)
	}
	err = syscall.Close(fd)
	if err != nil {
		return nil, err
	}

	locktype := LOCKFILE_EXCLUSIVE_LOCK

	if ro {
		locktype = 0
	}

	return &lockfile{
		stateMutex: &sync.Mutex{},
		rwMutex:    &sync.RWMutex{},
		file:       path,
		lw:         stringid.GenerateRandomID(),
		locktype:   locktype,
		locked:     false,
		ro:         ro}, nil
}

type lockfile struct {
	// rwMutex serializes concurrent reader-writer acquisitions in the same process space
	rwMutex *sync.RWMutex
	// stateMutex is used to synchronize concurrent accesses to the state below
	stateMutex *sync.Mutex
	counter    int64
	file       string
	fd         syscall.Handle
	lw         string
	locktype   uint32
	locked     bool
	ro         bool
	recursive  bool
}

func (l *lockfile) lock(lType uint32, recursive bool) {
	switch lType {
	case 0:
		l.rwMutex.RLock()
	case LOCKFILE_EXCLUSIVE_LOCK:
		if recursive {
			// NOTE: that's okay as recursive is only set in RecursiveLock(), so
			// there's no need to protect against hypothetical RDLCK cases.
			l.rwMutex.RLock()
		} else {
			l.rwMutex.Lock()
		}
	default:
		panic(fmt.Sprintf("attempted to acquire a file lock of unrecognized type %d", lType))
	}
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()
	if l.counter == 0 {
		// If we're the first reference on the lock, we need to open the file again.
		fd, err := openLock(l.file, l.ro)
		if err != nil {
			panic(fmt.Sprintf("error opening %q: %v", l.file, err))
		}
		l.fd = fd

		var ol syscall.Overlapped

		// Optimization: only use the (expensive) fcntl syscall when
		// the counter is 0.  In this case, we're either the first
		// reader lock or a writer lock.
		ret, _, err := procLockFileEx.Call(uintptr(l.fd), uintptr(lType), 0, uintptr(MAXDWORD), uintptr(MAXDWORD), uintptr(unsafe.Pointer(&ol)))
		if ret == 0 {
			return
		}
	}
	l.locktype = lType
	l.locked = true
	l.recursive = recursive
	l.counter++
}

// Lock locks the lockfile as a writer.  Panic if the lock is a read-only one.
func (l *lockfile) Lock() {
	if l.ro {
		panic("can't take write lock on read-only lock file")
	} else {
		l.lock(LOCKFILE_EXCLUSIVE_LOCK, false)
	}
}

func (l *lockfile) RecursiveLock() {
	if l.ro {
		l.RLock()
	} else {
		l.lock(LOCKFILE_EXCLUSIVE_LOCK, true)
	}
}

func (l *lockfile) RLock() {
	l.lock(0, false)
}

func (l *lockfile) Unlock() {
	l.stateMutex.Lock()
	if l.locked == false {
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
		_ = syscall.Close(l.fd)
	}
	if l.locktype == 0 || l.recursive {
		l.rwMutex.RUnlock()
	} else {
		l.rwMutex.Unlock()
	}
	l.stateMutex.Unlock()
}

// Locked checks if lockfile is locked for writing by a thread in this process.
func (l *lockfile) Locked() bool {
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()
	return l.locked && (l.locktype == LOCKFILE_EXCLUSIVE_LOCK)
}

func (l *lockfile) Modified() (bool, error) {
	l.stateMutex.Lock()
	id := make([]byte, len(l.lw))
	if !l.locked {
		panic("attempted to check last-writer in lockfile without locking it first")
	}
	defer l.stateMutex.Unlock()
	highoffest := int32(0)
	_, err := syscall.SetFilePointer(l.fd, 0, &highoffest, 0)
	if err != nil {
		return true, err
	}
	n, err := syscall.Read(l.fd, id)
	if err != nil {
		return true, err
	}
	if n != len(id) {
		return true, nil
	}
	lw := l.lw
	l.lw = string(id)
	return l.lw != lw, nil
}

func (l *lockfile) Touch() error {
	l.stateMutex.Lock()
	if !l.locked || (l.locktype != LOCKFILE_EXCLUSIVE_LOCK) {
		panic("attempted to update last-writer in lockfile without the write lock")
	}
	defer l.stateMutex.Unlock()
	l.lw = stringid.GenerateRandomID()
	id := []byte(l.lw)
	n, err := syscall.Write(l.fd, id)
	if err != nil {
		return err
	}
	if n != len(id) {
		return syscall.ENOSPC
	}
	return nil
}
func (l *lockfile) IsReadWrite() bool {
	return !l.ro
}

func (l *lockfile) TouchedSince(when time.Time) bool {
	stat, err := os.Stat(l.file)
	if err != nil {
		return true
	}
	return when.Before(stat.ModTime())
}
