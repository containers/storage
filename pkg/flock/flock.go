// flock exposes an API around flock(2) but allows for synchronizing within and
// across process spaces, making it suitable for currently running threads and
// processes alike. To support long-running daemons, the locks' file descriptors
// are closed when the lock is being released via Unlock() or RUnlock().
//
// Note that the flock package supports UNIX-only but compiles on Windows for
// compatibility reasons.

package flock

import (
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
)

var (
	errInternal       = errors.New("internal error: flock")
	errNotImplemented = errors.New("internal error: flock: not implemented")
)

// Flock implements a flock(2)-based file lock. It can be used to synchronize
// within the same process space and across process spaces.
type Flock interface {
	// Lock locks the exclusive Flock for writing.
	Lock() error
	// TryLock locks the exclusive Flock without blocking. The first return
	// value indicates if the Flock has been acquired.
	TryLock() (bool, error)
	// RLock locks the shared Flock for reading.  Note that RLock is busy
	// looping for acquiring the Flock.
	RLock() error
	// Unlock unlocks the Flock. If it's a shared Flock, unlock must be called
	// by the number of RLock() calls for the underlying lock file to be
	// unlocked.
	Unlock() error
	// Locked indicates if the Flock is locked.
	Locked() bool
}

// flockType indicates the type of the Flock.
type flockType int

const (
	// flockUnlocked indicates an unlock flock
	flockUnlocked flockType = iota
	// flockShared indicates a shared flock (i.e., reader)
	flockShared
	// flockExclusive indicates an exclusive flock (i.e., writer)
	flockExclusive
)

// flock implements the Flock interface
type flock struct {
	path string              // path to the lock file
	sem  *semaphore.Weighted // process-space internal flock implementation

	mutex     *sync.Mutex // synchronization of state below
	flockType flockType   // current type of the lock
	fd        int         // file descriptor
	refs      uint        // reference counting
}
