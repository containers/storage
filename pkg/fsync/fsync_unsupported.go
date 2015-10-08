// +build !linux

package fsync

import (
	"path/filepath"
	"sync"
)

// Mutex represents an RWMutex which synchronizes its state with a file lock,
// allowing two process which use the same lock file to share reading and
// writing locks.
type Mutex struct {
	rw       sync.RWMutex
	lockfile string
}

// RLock obtains a read lock on the Mutex.
func (m *Mutex) RLock() {
	m.rw.RLock()
}

// RUnlock releases a read lock on the Mutex.
func (m *Mutex) RUnlock() {
	m.rw.RUnlock()
}

// Lock obtains a write lock on the Mutex.
func (m *Mutex) Lock() {
	m.rw.Lock()
}

// Unlock releases a write lock on the Mutex.
func (m *Mutex) Unlock() {
	m.rw.Unlock()
}

// Touch updates the timestamp on the lock file to signal other processes or
// threads that the data protected by the lock has been modified by someone
// else.
func (m *Mutex) Touch() error {
	return nil
}

// Updated tells us if the timestamp on the lock file is more recent than the
// last time we recorded for the lockfile.  Should only be called with the lock
// held.
func (m *Mutex) Updated() bool {
	return false
}

// get initializes a mutex, opening the lockfile.
func get(lockfile string) (*Mutex, error) {
	lockMgr.m.Lock()
	defer lockMgr.m.Unlock()

	name, err := filepath.Abs(lockfile)
	if err != nil {
		return nil, err
	}
	name = filepath.Clean(name)
	fl, ok := lockMgr.locks[name]
	if !ok {
		fl = &Mutex{
			lockfile: name,
		}
		lockMgr.locks[name] = fl
	}
	return fl, nil
}
