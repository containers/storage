package directory

import (
	"os"
	"path/filepath"
)

// DiskUsage is a structure that describes the disk usage (size and inode count)
// of a particular directory.
type DiskUsage struct {
	Size       int64
	InodeCount int64
}

// MoveToSubdir moves all contents of a directory to a subdirectory underneath the original path
func MoveToSubdir(oldpath, subdir string) error {
	entries, err := os.ReadDir(oldpath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != subdir {
			oldName := filepath.Join(oldpath, entry.Name())
			newName := filepath.Join(oldpath, subdir, entry.Name())
			if err := os.Rename(oldName, newName); err != nil {
				return err
			}
		}
	}
	return nil
}
