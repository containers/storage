// +build linux

package fsync

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
)

type holderID [16]byte

// Mutex represents an RWMutex which synchronizes its state with a file lock,
// allowing two process which use the same lock file to share reading and
// writing locks.
type Mutex struct {
	rw               sync.RWMutex
	m                sync.Mutex
	lockfile         string
	locked           bool
	lockfd           int
	readers, writers int
	us, holder       holderID
}

// lockop obtains or clears a file lock on the specified descriptor, blocking
// and retrying indefinitely if it fails to do so for any reason.
func lockop(name string, fd, lockop int) {
	err := syscall.Flock(fd, lockop)
	for err != nil {
		logrus.Debugf("waiting for file lock on %s", name)
		time.Sleep(100 * time.Millisecond)
		err = syscall.Flock(fd, lockop)
		if err == nil {
			logrus.Debugf("obtained file lock on %s", name)
		}
	}
}

// RLock obtains a read lock on the Mutex.
func (m *Mutex) RLock() {
	m.rw.RLock()
	m.m.Lock()
	defer m.m.Unlock()
	m.readers++
	if m.readers == 1 {
		lockop(m.lockfile, m.lockfd, syscall.LOCK_SH)
		m.locked = true
	}
}

// RUnlock releases a read lock on the Mutex.
func (m *Mutex) RUnlock() {
	m.rw.RUnlock()
	m.m.Lock()
	defer m.m.Unlock()
	m.readers--
	if m.readers == 0 {
		if !m.locked {
			panic(fmt.Sprintf("attempted to unlock %s while not locked", m.lockfile))
		}
		lockop(m.lockfile, m.lockfd, syscall.LOCK_UN)
		m.locked = false
	}
}

// Lock obtains a write lock on the Mutex.
func (m *Mutex) Lock() {
	m.rw.Lock()
	m.m.Lock()
	defer m.m.Unlock()
	m.writers++
	if m.writers == 1 {
		lockop(m.lockfile, m.lockfd, syscall.LOCK_EX)
		m.locked = true
	}
}

// Unlock releases a write lock on the Mutex.
func (m *Mutex) Unlock() {
	m.rw.Unlock()
	m.m.Lock()
	defer m.m.Unlock()
	m.writers--
	if m.writers == 0 {
		if !m.locked {
			panic(fmt.Sprintf("attempted to unlock %s while not locked", m.lockfile))
		}
		lockop(m.lockfile, m.lockfd, syscall.LOCK_UN)
		m.locked = false
	}
}

// Touch updates the contents of the lock file to hold our ID, as an indicator
// that other processes can consult to see that it was this process that last
// modified the data that's protected by the lock.  Should only be called with
// a write lock held.
func (m *Mutex) Touch() error {
	n, err := rand.Read(m.us[:])
	if n != len(m.us) || err != nil {
		return err
	}
	if _, err := syscall.Pwrite(m.lockfd, m.us[:], 0); err != nil {
		return err
	}
	m.holder = m.us
	return nil
}

// Updated tells us if the last recorded ID in the lock file is different from
// ours and has changed since we last looked at it.  Should only be called with
// either a read or write lock held.
func (m *Mutex) Updated() bool {
	var holder holderID
	if n, err := syscall.Pread(m.lockfd, holder[:], 0); err != nil && n != 0 {
		return true
	}
	updated := bytes.Compare(holder[:], m.us[:]) != 0 &&
		bytes.Compare(holder[:], m.holder[:]) != 0
	m.holder = holder
	return updated
}

// get initializes a mutex, opening the lockfile.
func get(lockfile string) (*Mutex, error) {
	var noHolder, us holderID

	lockMgr.m.Lock()
	defer lockMgr.m.Unlock()

	if bytes.Compare(us[:], noHolder[:]) == 0 {
		n, err := rand.Read(us[:])
		if n != len(us) || err != nil {
			return nil, err
		}
	}

	name, err := filepath.Abs(lockfile)
	if err != nil {
		return nil, err
	}
	name = filepath.Clean(name)
	fl, ok := lockMgr.locks[name]
	if !ok {
		lockfd, err := syscall.Open(name, syscall.O_CREAT|syscall.O_RDWR, syscall.S_IRUSR|syscall.S_IWUSR)
		if err != nil {
			return nil, err
		}
		syscall.CloseOnExec(lockfd)
		fl = &Mutex{
			lockfd:   lockfd,
			lockfile: name,
			us:       us,
		}
		lockMgr.locks[name] = fl
	}
	return fl, nil
}
