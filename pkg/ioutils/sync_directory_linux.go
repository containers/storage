//go:build linux

package ioutils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// SyncDirectoryContents flushes file data and directory metadata under dir to
// physical storage. Call this before atomically renaming a fully populated
// staging directory to its final location.
func SyncDirectoryContents(dir string) error {
	var dirs []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}

		syncErr := unix.Fdatasync(int(f.Fd()))
		closeErr := f.Close()
		if syncErr != nil {
			return syncErr
		}

		return closeErr
	})
	if err != nil {
		return fmt.Errorf("sync directory contents in %q: %w", dir, err)
	}

	for i := len(dirs) - 1; i >= 0; i-- {
		dfd, err := os.Open(dirs[i])
		if err != nil {
			return fmt.Errorf("open directory %q for sync: %w", dirs[i], err)
		}

		syncErr := unix.Fsync(int(dfd.Fd()))
		closeErr := dfd.Close()
		if syncErr != nil {
			return fmt.Errorf("sync directory %q: %w", dirs[i], syncErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close directory %q after sync: %w", dirs[i], closeErr)
		}
	}

	return nil
}
