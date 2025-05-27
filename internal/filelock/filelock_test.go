package filelock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenLock(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		prepare func() (path string, readOnly bool)
	}{
		{
			name: "file exists (read/write)",
			prepare: func() (string, bool) {
				tempFile, err := os.CreateTemp("", "lock-")
				require.NoError(t, err)
				tempFile.Close()
				return tempFile.Name(), false
			},
		},
		{
			name: "file exists readonly (readonly)",
			prepare: func() (string, bool) {
				tempFile, err := os.CreateTemp("", "lock-")
				require.NoError(t, err)
				tempFile.Close()
				return tempFile.Name(), true
			},
		},
		{
			name: "base dir exists (read/write)",
			prepare: func() (string, bool) {
				tempDir := os.TempDir()
				require.DirExists(t, tempDir)
				return filepath.Join(tempDir, "test-1.lock"), false
			},
		},
		{
			name: "base dir not exists (read/write)",
			prepare: func() (string, bool) {
				tempDir, err := os.MkdirTemp("", "lock-")
				require.NoError(t, err)
				return filepath.Join(tempDir, "subdir", "test-1.lock"), false
			},
		},
	} {
		path, readOnly := tc.prepare()

		fd, err := OpenLock(path, readOnly)
		require.NoError(t, err, tc.name)
		UnlockAndCloseHandle(fd)

		fd, err = OpenLock(path, readOnly)
		require.NoError(t, err)
		UnlockAndCloseHandle(fd)

		require.Nil(t, os.RemoveAll(path))
	}
}

func TestOpenLockCreatesParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "subdir", "lockfile")
	fd, err := OpenLock(lockPath, false)
	require.NoError(t, err)
	UnlockAndCloseHandle(fd)
}

func TestTryLockFileAndLockFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "lockfile")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	fd, err := OpenLock(tmpFile.Name(), false)
	require.NoError(t, err)
	defer UnlockAndCloseHandle(fd)
	require.NoError(t, TryLockFile(fd, WriteLock))
	UnlockAndCloseHandle(fd)

	fd2, err := OpenLock(tmpFile.Name(), false)
	require.NoError(t, err)
	require.NoError(t, LockFile(fd2, WriteLock))
	UnlockAndCloseHandle(fd2)
}

func TestCloseHandleIdempotent(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "lockfile")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	fd, err := OpenLock(tmpFile.Name(), false)
	require.NoError(t, err)
	CloseHandle(fd)
	CloseHandle(fd)
}
