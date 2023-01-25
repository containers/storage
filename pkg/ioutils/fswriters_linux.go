package ioutils

import (
	"os"

	"golang.org/x/sys/unix"
)

func fdatasync(f *os.File) error {
	return unix.Fdatasync(int(f.Fd()))
}

func swapOrMove(oldpath string, newpath string) error {
	err := unix.Renameat2(unix.AT_FDCWD, oldpath, unix.AT_FDCWD, newpath, unix.RENAME_EXCHANGE)
	if err != nil {
		// unlikely that rename will succeed if renameat2 failed, but just in case
		err = os.Rename(oldpath, newpath)
	}
	return err
}
