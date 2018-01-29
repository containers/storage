package graphdriver

import (
	"os"
	"syscall"
)

// chrootOrChdir() is either a chdir() to the specified path, or a chroot() to the
// specified path followed by chdir() to the new root directory
func chrootOrChdir(path string) error {
	if err := syscall.Chdir(path); err != nil {
		fmt.Printf("error changing to %q: %v", os.Args[1], err)
		os.Exit(1)
	}
	return nil
}
