// +build !unix

package flock

func New(path string) (flock, error) {
	return flock{}, errNotImplemented
}

func (f *flock) Locked() bool {
	return false
}

func (f *flock) Lock() error {
	return errNotImplemented
}

func (f *flock) TryLock() (bool, error) {
	return false, errNotImplemented
}

func (f *flock) RLock() error {
	return errNotImplemented
}

func (f *flock) Unlock() error {
	return errNotImplemented
}
