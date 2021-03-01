package types

import (
	"fmt"
	"os"
	"testing"

	"gotest.tools/assert"
)

func TestGetRootlessStorageOpts(t *testing.T) {
	envDriver, envDriverSet := os.LookupEnv("STORAGE_DRIVER")
	os.Unsetenv("STORAGE_DRIVER")

	const vfsDriver = "vfs"
	const overlayDriver = "overlay"

	t.Run("systemDriver=btrfs", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = "btrfs"
		storageOpts, err := getRootlessStorageOpts(1000, systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, "btrfs")
	})

	t.Run("systemDriver=overlay", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = overlayDriver
		storageOpts, err := getRootlessStorageOpts(1000, systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, overlayDriver)
	})

	t.Run("systemDriver=overlay2", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = "overlay2"
		storageOpts, err := getRootlessStorageOpts(1000, systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, "overlay2")
	})

	t.Run("systemDriver=vfs", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = vfsDriver
		storageOpts, err := getRootlessStorageOpts(1000, systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, vfsDriver)
	})

	t.Run("systemDriver=aufs", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = "aufs"
		storageOpts, err := getRootlessStorageOpts(1000, systemOpts)
		assert.NilError(t, err)
		assert.Assert(t, storageOpts.GraphDriverName == overlayDriver || storageOpts.GraphDriverName == vfsDriver, fmt.Sprintf("The rootless driver should be set to 'overlay' or 'vfs' not '%v'", storageOpts.GraphDriverName))
	})

	t.Run("systemDriver=devmapper", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = "devmapper"
		storageOpts, err := getRootlessStorageOpts(1000, systemOpts)
		assert.NilError(t, err)
		assert.Assert(t, storageOpts.GraphDriverName == overlayDriver || storageOpts.GraphDriverName == vfsDriver, fmt.Sprintf("The rootless driver should be set to 'overlay' or 'vfs' not '%v'", storageOpts.GraphDriverName))
	})

	t.Run("systemDriver=zfs", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = "zfs"
		storageOpts, err := getRootlessStorageOpts(1000, systemOpts)
		assert.NilError(t, err)
		assert.Assert(t, storageOpts.GraphDriverName == overlayDriver || storageOpts.GraphDriverName == vfsDriver, fmt.Sprintf("The rootless driver should be set to 'overlay' or 'vfs' not '%v'", storageOpts.GraphDriverName))
	})

	t.Run("STORAGE_DRIVER=btrfs", func(t *testing.T) {
		os.Setenv("STORAGE_DRIVER", "btrfs")
		defer os.Unsetenv("STORAGE_DRIVER")
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = vfsDriver
		storageOpts, err := getRootlessStorageOpts(1000, systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, "btrfs")
	})

	t.Run("STORAGE_DRIVER=zfs", func(t *testing.T) {
		os.Setenv("STORAGE_DRIVER", "zfs")
		defer os.Unsetenv("STORAGE_DRIVER")
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = vfsDriver
		storageOpts, err := getRootlessStorageOpts(1000, systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, "zfs")
	})

	if envDriverSet {
		os.Setenv("STORAGE_DRIVER", envDriver)
	} else {
		os.Unsetenv("STORAGE_DRIVER")
	}
}
