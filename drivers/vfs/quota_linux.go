package vfs

import (
	"github.com/containers/storage/drivers/quota"
	"github.com/sirupsen/logrus"
)

type driverQuota struct {
	quotaCtl *quota.Control
}

func setupDriverQuota(driver *Driver) {
	if quotaCtl, err := quota.NewControl(driver.homes[0]); err == nil {
		driver.quotaCtl = quotaCtl
	} else if err != quota.ErrQuotaNotSupported {
		logrus.Warnf("Unable to setup quota: %v\n", err)
	}
}

func (d *Driver) setupQuota(dir string, size uint64) error {
	return d.quotaCtl.SetQuota(dir, quota.Quota{Size: size})
}

func (d *Driver) quotaSupported() bool {
	return d.quotaCtl != nil
}
