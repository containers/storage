//go:build linux && composefs && cgo
// +build linux,composefs,cgo

package overlay

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/containers/storage/pkg/loopback"
	"golang.org/x/sys/unix"
)

var (
	composeFsHelperOnce sync.Once
	composeFsHelperPath string
	composeFsHelperErr  error
)

func getComposeFsHelper() (string, error) {
	composeFsHelperOnce.Do(func() {
		composeFsHelperPath, composeFsHelperErr = exec.LookPath("composefs-from-json")
	})
	return composeFsHelperPath, composeFsHelperErr
}

func composeFsSupported() bool {
	_, err := getComposeFsHelper()
	return err == nil
}

func generateComposeFsBlob(toc []byte, destFile string) error {
	writerJson, err := getComposeFsHelper()
	if err != nil {
		return fmt.Errorf("failed to find composefs-from-json: %w", err)
	}

	fd, err := unix.Openat(unix.AT_FDCWD, destFile, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC|unix.O_EXCL|unix.O_CLOEXEC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	outFd := os.NewFile(uintptr(fd), "outFd")

	defer outFd.Close()
	cmd := exec.Command(writerJson, "--format=erofs", "--out=/proc/self/fd/3", "/proc/self/fd/0")
	cmd.ExtraFiles = []*os.File{outFd}
	cmd.Stdin = bytes.NewReader(toc)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert json to erofs: %w", err)
	}
	return nil
}

func mountErofsBlob(blobFile, mountPoint string) error {
	loop, err := loopback.AttachLoopDevice(blobFile)
	if err != nil {
		return err
	}
	defer loop.Close()

	return unix.Mount(loop.Name(), mountPoint, "erofs", unix.MS_RDONLY, "ro")
}
