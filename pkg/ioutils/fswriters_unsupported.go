//go:build !linux
// +build !linux

package ioutils

import (
	"os"
)

func fdatasync(f *os.File) error {
	return f.Sync()
}

func swapOrMove(oldpath string, newpath string) error {
	return os.Rename(oldpath, newpath)
}
