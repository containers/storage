// +build !linux

package vfs // import "github.com/containers/storage/daemon/graphdriver/vfs"

import "github.com/containers/storage/daemon/graphdriver/quota"

type driverQuota struct {
}

func setupDriverQuota(driver *Driver) error {
	return nil
}

func (d *Driver) setupQuota(dir string, size uint64) error {
	return quota.ErrQuotaNotSupported
}

func (d *Driver) quotaSupported() bool {
	return false
}
