package archive

import (
	"golang.org/x/sys/unix"
)

// resetSymlinkTimes sets the atime and mtime of a symlink to a known value, to test
// whether changes to the symlink target are detected correctly.
func resetSymlinkTimes(path string) error {
	ts := []unix.Timeval{unix.NsecToTimeval(0), unix.NsecToTimeval(0)}
	return unix.Lutimes(path, ts)
}
