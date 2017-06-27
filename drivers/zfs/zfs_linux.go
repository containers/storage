package zfs

import (
	"fmt"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	"github.com/containers/storage/drivers"
)

func checkRootdirFs(rootdir string) error {
	var buf syscall.Statfs_t
	if err := syscall.Statfs(rootdir, &buf); err != nil {
		return fmt.Errorf("Failed to access '%s': %s", rootdir, err)
	}

	if graphdriver.FsMagic(buf.Type) != graphdriver.FsMagicZfs {
		logrus.Debugf("[zfs] no zfs dataset found for rootdir '%s'", rootdir)
		return errors.Wrapf(graphdriver.ErrPrerequisites, "%q is not on a zfs filesystem", rootdir)
	}

	return nil
}

func getMountpoint(id string) string {
	return id
}
