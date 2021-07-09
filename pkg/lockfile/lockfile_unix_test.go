// +build linux solaris darwin freebsd

package lockfile

import (
	"io/ioutil"
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
				tempFile, err := ioutil.TempFile("", "lock-")
				require.NoError(t, err)
				return tempFile.Name(), false
			},
		},
		{
			name: "file exists readonly (readonly)",
			prepare: func() (string, bool) {
				tempFile, err := ioutil.TempFile("", "lock-")
				require.NoError(t, err)
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
				tempDir, err := ioutil.TempDir("", "lock-")
				require.NoError(t, err)
				return filepath.Join(tempDir, "subdir", "test-1.lock"), false
			},
		},
	} {
		path, readOnly := tc.prepare()

		_, err := openLock(path, readOnly)

		require.NoError(t, err, tc.name)

		_, err = openLock(path, readOnly)
		require.NoError(t, err)

		require.Nil(t, os.RemoveAll(path))
	}
}
