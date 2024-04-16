package types

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/containers/storage/pkg/idtools"
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
		if canUseRootlessOverlay(home, runhome) {
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

func TestSetRemapUIDsGIDsOpts(t *testing.T) {
	var remapOpts StoreOptions
	uidmap := []idtools.IDMap{
		{
			ContainerID: 0,
			HostID:      1000000000,
			Size:        30000,
		},
	}
	gidmap := []idtools.IDMap{
		{
			ContainerID: 0,
			HostID:      1500000000,
			Size:        60000,
		},
	}

	err := ReloadConfigurationFile("./storage_test.conf", &remapOpts)
	require.NoError(t, err)
	if !reflect.DeepEqual(uidmap, remapOpts.UIDMap) {
		t.Errorf("Failed to set UIDMap: Expected %v Actual %v", uidmap, remapOpts.UIDMap)
	}
	if !reflect.DeepEqual(gidmap, remapOpts.GIDMap) {
		t.Errorf("Failed to set GIDMap: Expected %v Actual %v", gidmap, remapOpts.GIDMap)
	}
}

func TestSetRemapUserGroupOpts(t *testing.T) {
	var remapOpts StoreOptions

	user := os.Getenv("USER")
	if user == "root" {
		t.Skip("This test is enabled only rootless user")
	}

	configPath := "./remap_user_test.conf"
	config := fmt.Sprintf(`
[storage]
driver = ""

[storage.options]
remap-uids = "0:1000000000:30000"
remap-gids = "0:1500000000:60000"

remap-user = "%s"
remap-group = "%s"
`, user, user)
	f, err := os.Create(configPath)
	require.NoError(t, err)
	defer func() {
		f.Close()
		os.Remove(configPath)
	}()

	_, err = f.Write([]byte(config))
	require.NoError(t, err)

	mappings, err := idtools.NewIDMappings(user, user)
	require.NoError(t, err)
	err = ReloadConfigurationFile(configPath, &remapOpts)
	require.NoError(t, err)
	if !reflect.DeepEqual(mappings.UIDs(), remapOpts.UIDMap) {
		t.Errorf("Failed to set UIDMap: Expected %v Actual %v", mappings.UIDs(), remapOpts.UIDMap)
	}
	if !reflect.DeepEqual(mappings.GIDs(), remapOpts.GIDMap) {
		t.Errorf("Failed to set GIDMap: Expected %v Actual %v", mappings.GIDs(), remapOpts.GIDMap)
	}
}

func TestReloadConfigurationFile(t *testing.T) {
	content := bytes.NewBufferString("")
	logrus.SetOutput(content)
	var storageOpts StoreOptions
	err := ReloadConfigurationFile("./storage_broken.conf", &storageOpts)
	require.NoError(t, err)
	assert.Equal(t, storageOpts.RunRoot, "/run/containers/test")
	logrus.SetOutput(os.Stderr)

	assert.Equal(t, strings.Contains(content.String(), "Failed to decode the keys [\\\"foo\\\" \\\"storage.options.graphroot\\\"] from \\\"./storage_broken.conf\\\"\""), true)
}
