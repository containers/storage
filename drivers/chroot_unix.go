// +build linux darwin freebsd solaris

package graphdriver

import (
	"fmt"
	"os"
	"syscall"
)

// chrootOrChdir() is either a chdir() to the specified path, or a chroot() to the
// specified path followed by chdir() to the new root directory
func chrootOrChdir(path string) error {
	if err := syscall.Chroot(os.Args[1]); err != nil {
		fmt.Printf("error chrooting to %q: %v", os.Args[1], err)
		os.Exit(1)
	}
	if err := syscall.Chdir(string(os.PathSeparator)); err != nil {
		fmt.Printf("error changing to %q: %v", os.Args[1], err)
		os.Exit(1)
	}
	return nil
}
