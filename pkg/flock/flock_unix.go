// +build linux solaris darwin freebsd

package flock

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sys/unix"
)

// New returns a Flock for path.
func New(path string) (flock, error) {
	// Make sure we can operate on the path.
	fd, err := unix.Open(path, os.O_RDWR|os.O_CREATE, unix.S_IRUSR|unix.S_IWUSR)
	if err != nil {
		return flock{}, err
	}
	if err := unix.Close(fd); err != nil {
		return flock{}, err
	}
	return flock{
		flockType: flockUnlocked,
		fd:        -1,
		path:      path,
		mutex:     new(sync.Mutex),
		sem:       semaphore.NewWeighted(1),
	}, nil
}

// open opens the flock's path and sets the file descriptor. State must be
// synchronized by the caller.
func (f *flock) open() error {
	if f.fd != -1 || f.flockType != flockUnlocked {
		return fmt.Errorf("%v: trying to open improper Flock %q %d %d", errInternal, f.path, f.fd, f.flockType)
	}
	fd, err := unix.Open(f.path, os.O_RDWR|os.O_CREATE, unix.S_IRUSR|unix.S_IWUSR)
	if err != nil {
		return err
	}
	f.fd = fd
	return nil
}

func (f *flock) Locked() bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.refs > 0
}

func (f *flock) Lock() (err error) {
	if err := f.sem.Acquire(context.Background(), 1); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			return
		}
		f.mutex.Lock()
		f.refs++
		f.flockType = flockExclusive
		f.mutex.Unlock()
	}()

	f.mutex.Lock()
	if err := f.open(); err != nil {
		f.mutex.Unlock()
		return err
	}
	f.mutex.Unlock()

	return unix.Flock(f.fd, unix.LOCK_EX)
}

func (f *flock) TryLock() (acquired bool, err error) {
	if acquired := f.sem.TryAcquire(1); !acquired {
		return false, nil
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	defer func() { // Deferred clean-ups.
		if err == nil {
			// No error, so we acquired the lock.
			f.refs++
			acquired = true
			f.flockType = flockExclusive
			return
		}
		if errors.Cause(err) == unix.EWOULDBLOCK {
			// If it's a blocking error, nil the error.
			err = nil
		}
		// Now, we still need to clean-up and close the file descriptor.
		if closeErr := unix.Close(f.fd); closeErr != nil {
			// Note: errors.Wrap*(nil, ...) is always nil
			if err == nil {
				err = closeErr
			} else {
				errors.Wrap(err, closeErr.Error())
			}
		}
		f.fd = -1
		f.sem.Release(1)
	}()

	if err := f.open(); err != nil {
		return false, err
	}
	return acquired, unix.Flock(f.fd, unix.LOCK_EX|unix.LOCK_NB)
}

func (f *flock) RLock() error {
	for { // Busy loop to avoid dead locks.
		f.mutex.Lock()
		// Unlock requires owning the mutex, so we will increment the ref conter
		// **before** any concurrent threads may unlock underlying file.
		if f.flockType == flockShared {
			break
		}
		// If we don't already own the flock (see upper case), we fight with
		// other threads for the semaphore.
		if f.sem.TryAcquire(1) {
			break
		}
		f.mutex.Unlock()
	}

	f.refs++
	if f.refs > 1 {
		f.mutex.Unlock()
		return nil
	}

	if err := f.open(); err != nil {
		f.mutex.Unlock()
		return err
	}

	f.flockType = flockShared
	f.mutex.Unlock()
	return unix.Flock(f.fd, unix.LOCK_SH)
}

func (f *flock) Unlock() error {
	// Note: it's crucial we own the mutex for the entire unlock() procedure.
	// RLock() depends on it.
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.refs == 0 {
		return fmt.Errorf("%v: unlocking unlocked Flock %q", errInternal, f.path)
	}

	f.refs--
	if f.refs == 0 {
		defer func() {
			f.flockType = flockUnlocked
			f.fd = -1
			f.sem.Release(1)
		}()
		if err := unix.Flock(f.fd, unix.LOCK_UN); err != nil {
			return err
		}
		if err := unix.Close(f.fd); err != nil {
			return err
		}
	}
	return nil
}
