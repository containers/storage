//go:build freebsd
// +build freebsd

package system

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func Lchflags(path string, flags uint32) error {
	p, err := unix.BytePtrFromString(path)
	if err != nil {
		return err
	}
	_, _, e1 := unix.Syscall(unix.SYS_LCHFLAGS, uintptr(unsafe.Pointer(p)), uintptr(flags), 0)
	if e1 != 0 {
		return e1
	}
	return nil
}
