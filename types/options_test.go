package types

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containers/storage/pkg/unshare"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestGetRootlessStorageOpts(t *testing.T) {
	envDriver, envDriverSet := os.LookupEnv("STORAGE_DRIVER")
	os.Unsetenv("STORAGE_DRIVER")

	const vfsDriver = "vfs"

	t.Run("systemDriver=<unset>", func(t *testing.T) {
		systemOpts := StoreOptions{}

		td := t.TempDir()
		home := filepath.Join(td, "unset-driver-home")
		runhome := filepath.Join(td, "unset-driver-runhome")
		defer os.RemoveAll(home)
		defer os.RemoveAll(runhome)

		systemOpts.GraphRoot = home
		systemOpts.RunRoot = runhome
		storageOpts, err := getRootlessStorageOpts(systemOpts)

		assert.NilError(t, err)
		expectedDriver := vfsDriver
		if canUseRootlessOverlay() {
			expectedDriver = overlayDriver
		}
		assert.Equal(t, storageOpts.GraphDriverName, expectedDriver)
	})

	t.Run("systemDriver=btrfs", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = "btrfs"
		storageOpts, err := getRootlessStorageOpts(systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, "btrfs")
	})

	t.Run("systemDriver=overlay", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = overlayDriver
		storageOpts, err := getRootlessStorageOpts(systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, overlayDriver)
	})

	t.Run("systemDriver=overlay2", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = "overlay2"
		storageOpts, err := getRootlessStorageOpts(systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, overlayDriver)
	})

	t.Run("systemDriver=vfs", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = vfsDriver
		storageOpts, err := getRootlessStorageOpts(systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, vfsDriver)
	})

	t.Run("systemDriver=aufs", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = "aufs"
		storageOpts, err := getRootlessStorageOpts(systemOpts)
		assert.NilError(t, err)
		assert.Assert(t, storageOpts.GraphDriverName == overlayDriver || storageOpts.GraphDriverName == vfsDriver, fmt.Sprintf("The rootless driver should be set to 'overlay' or 'vfs' not '%v'", storageOpts.GraphDriverName))
	})

	t.Run("systemDriver=zfs", func(t *testing.T) {
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = "zfs"
		storageOpts, err := getRootlessStorageOpts(systemOpts)
		assert.NilError(t, err)
		assert.Assert(t, storageOpts.GraphDriverName == overlayDriver || storageOpts.GraphDriverName == vfsDriver, fmt.Sprintf("The rootless driver should be set to 'overlay' or 'vfs' not '%v'", storageOpts.GraphDriverName))
	})

	t.Run("STORAGE_DRIVER=btrfs", func(t *testing.T) {
		t.Setenv("STORAGE_DRIVER", "btrfs")
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = vfsDriver
		storageOpts, err := getRootlessStorageOpts(systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, "btrfs")
	})

	t.Run("STORAGE_DRIVER=zfs", func(t *testing.T) {
		t.Setenv("STORAGE_DRIVER", "zfs")
		systemOpts := StoreOptions{}
		systemOpts.GraphDriverName = vfsDriver
		storageOpts, err := getRootlessStorageOpts(systemOpts)
		assert.NilError(t, err)
		assert.Equal(t, storageOpts.GraphDriverName, "zfs")
	})

	if envDriverSet {
		os.Setenv("STORAGE_DRIVER", envDriver)
	} else {
		os.Unsetenv("STORAGE_DRIVER")
	}
}

func TestGetRootlessStorageOpts2(t *testing.T) {
	opts := StoreOptions{
		RootlessStoragePath: "/$HOME/$UID/containers/storage",
	}
	expectedPath := filepath.Join(os.Getenv("HOME"), fmt.Sprintf("%d", unshare.GetRootlessUID()), "containers/storage")
	storageOpts, err := getRootlessStorageOpts(opts)
	assert.NilError(t, err)
	assert.Equal(t, storageOpts.GraphRoot, expectedPath)
}

func TestReloadConfigurationFile(t *testing.T) {
	t.Run("broken", func(t *testing.T) {
		content := bytes.NewBufferString("")
		logrus.SetOutput(content)
		var storageOpts StoreOptions
		err := ReloadConfigurationFile("./storage_broken.conf", &storageOpts)
		require.NoError(t, err)
		assert.Equal(t, storageOpts.RunRoot, "/run/containers/test")
		logrus.SetOutput(os.Stderr)
		assert.Equal(t, strings.Contains(content.String(), "Failed to decode the keys [\\\"foo\\\" \\\"storage.options.graphroot\\\"] from \\\"./storage_broken.conf\\\"\""), true)
	})
	t.Run("imagestore-empty", func(t *testing.T) {
		expectedStore := ""
		expectedAdditionalStores := ""
		var storageOpts StoreOptions
		err := ReloadConfigurationFile("./storage_test.conf", &storageOpts)
		require.NoError(t, err)
		var actualStore, actualAdditionalStores string
		for _, o := range storageOpts.GraphDriverOptions {
			option := strings.Split(o, "=")
			switch option[0] {
			case storageOpts.GraphDriverName + ".imagestore":
				actualStore = option[1]
			case storageOpts.GraphDriverName + ".additionalimagestores":
				actualAdditionalStores = option[1]
			}
		}
		assert.Equal(t, actualStore, expectedStore)
		assert.Equal(t, actualAdditionalStores, expectedAdditionalStores)
	})
	t.Run("imagestore-many", func(t *testing.T) {
		expectedStore := "/var/lib/containers/storage1"
		expectedAdditionalStores := "/var/lib/containers/storage1,/var/lib/containers/storage2"
		var storageOpts StoreOptions
		err := ReloadConfigurationFile("./storage_imagestores_test.conf", &storageOpts)
		require.NoError(t, err)
		var actualStore, actualAdditionalStores string
		for _, o := range storageOpts.GraphDriverOptions {
			option := strings.Split(o, "=")
			switch option[0] {
			case storageOpts.GraphDriverName + ".imagestore":
				actualStore = option[1]
			case storageOpts.GraphDriverName + ".additionalimagestores":
				actualAdditionalStores = option[1]
			}
		}
		assert.Equal(t, actualStore, expectedStore)
		assert.Equal(t, actualAdditionalStores, expectedAdditionalStores)
	})
}
