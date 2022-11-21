package storage

import (
	"github.com/containers/storage/pkg/lockfile"
)

type Locker = lockfile.Locker

// Deprecated: Use lockfile.GetLockFile
func GetLockfile(path string) (lockfile.Locker, error) {
	return lockfile.GetLockfile(path)
}

// Deprecated: Use lockfile.GetROLockFile
func GetROLockfile(path string) (lockfile.Locker, error) {
	return lockfile.GetROLockfile(path)
}
