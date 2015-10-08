package fsync

import (
	"sync"
)

// rmutex is a wrapper for a Mutex which provides Lock and Unlock methods which
// merely call the underlying Mutex's RLock and RUnlock methods.
type rmutex struct {
	m *Mutex
}

type RLocker interface {
	sync.Locker
	Updated() bool
}

// RLocker returns a Locker which obtains and releases read locks on the underlying Mutex.
func (m *Mutex) RLocker() RLocker {
	return &rmutex{m: m}
}

// Lock obtains a read lock on the underlying Mutex.
func (r *rmutex) Lock() {
	r.m.RLock()
}

// Unlock releases a read lock on the underlying Mutex.
func (r *rmutex) Unlock() {
	r.m.RUnlock()
}

// Updated checks if the data protected by the lock was changed since we last
// modified it.
func (r *rmutex) Updated() bool {
	return r.m.Updated()
}

var lockMgr struct {
	m     sync.Mutex
	locks map[string]*Mutex
}

func init() {
	lockMgr.locks = make(map[string]*Mutex)
}

// Get returns a mutex which is tied to a lock on the specified lockfile, or nil on error.  The file descriptor is kept open.
func Get(lockfile string) (*Mutex, error) {
	m, err := get(lockfile)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// RLock obtains a read lock on the specified lock file, or returns an error.
func RLock(lockfile string) error {
	fl, err := Get(lockfile)
	if err != nil {
		return err
	}
	fl.RLock()
	return nil
}

// RUnlock releases a read lock on the specified lock file, or returns an error.
func RUnlock(lockfile string) error {
	fl, err := Get(lockfile)
	if err != nil {
		return err
	}
	fl.RUnlock()
	return nil
}

// Lock obtains a write lock on the specified lock file, or returns an error.
func Lock(lockfile string) error {
	fl, err := Get(lockfile)
	if err != nil {
		return err
	}
	fl.Lock()
	return nil
}

// Unlock releases a write lock on the specified lock file, or returns an error.
func Unlock(lockfile string) error {
	fl, err := Get(lockfile)
	if err != nil {
		return err
	}
	fl.Unlock()
	return nil
}
