//go:build linux || freebsd
// +build linux freebsd

package graphtest

import (
	"os"
	"path"
	"syscall"
	"testing"

	graphdriver "github.com/containers/storage/drivers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func verifyFile(t testing.TB, path string, mode os.FileMode, uid, gid uint32) {
	fi, err := os.Stat(path)
	require.NoError(t, err)

	actual := fi.Mode()
	assert.Equal(t, mode&os.ModeType, actual&os.ModeType, path)
	assert.Equal(t, mode&os.ModePerm, actual&os.ModePerm, path)
	assert.Equal(t, mode&os.ModeSticky, actual&os.ModeSticky, path)
	assert.Equal(t, mode&os.ModeSetuid, actual&os.ModeSetuid, path)
	assert.Equal(t, mode&os.ModeSetgid, actual&os.ModeSetgid, path)

	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		assert.Equal(t, uid, stat.Uid, path)
		assert.Equal(t, gid, stat.Gid, path)
	}
}

func createBase(t testing.TB, driver graphdriver.Driver, name string) {
	// We need to be able to set any perms
	oldmask := unix.Umask(0)
	defer unix.Umask(oldmask)

	err := driver.CreateReadWrite(name, "", nil)
	require.NoError(t, err)
	t.Cleanup(func() { removeLayer(t, driver, name) })

	dir, err := driver.Get(name, graphdriver.MountOpts{})
	require.NoError(t, err)
	defer func() {
		err = driver.Put(name)
		require.NoError(t, err)
	}()

	subdir := path.Join(dir, "a subdir")
	require.NoError(t, os.Mkdir(subdir, defaultSubdirPerms|os.ModeSticky))
	require.NoError(t, os.Chown(subdir, defaultSubdirOwner, defaultSubdirGroup))

	file := path.Join(dir, "a file")
	err = os.WriteFile(file, []byte("Some data"), defaultFilePerms|os.ModeSetuid)
	require.NoError(t, err)
}

func verifyBase(t testing.TB, driver graphdriver.Driver, name string, defaultPerm os.FileMode) {
	dir, err := driver.Get(name, graphdriver.MountOpts{})
	require.NoError(t, err)
	defer func() {
		err = driver.Put(name)
		require.NoError(t, err)
	}()

	verifyFile(t, dir, defaultPerm|os.ModeDir, 0, 0)

	subdir := path.Join(dir, "a subdir")
	verifyFile(t, subdir, defaultSubdirPerms|os.ModeDir|os.ModeSticky, defaultSubdirOwner, defaultSubdirGroup)

	file := path.Join(dir, "a file")
	verifyFile(t, file, defaultFilePerms|os.ModeSetuid, 0, 0)

	files, err := readDir(dir)
	require.NoError(t, err)
	assert.Len(t, files, 2)
}
