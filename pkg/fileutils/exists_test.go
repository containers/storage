package fileutils

import (
	"os"
	"path"
	"runtime"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExist(t *testing.T) {
	tempDir := t.TempDir()

	symlinkPath := path.Join(tempDir, "sl-working")
	danglingSymlinkPath := path.Join(tempDir, "sl-broken")

	err := os.Symlink(tempDir, symlinkPath)
	require.NoError(t, err)

	err = os.Symlink("fooobar123", danglingSymlinkPath)
	require.NoError(t, err)

	assertSameError := func(err1, err2 error, description string) {
		assert.Equal(t, err1 == nil, err2 == nil, description+": only one error is set")
		if err1 == nil {
			return
		}

		var pathErr1 *os.PathError
		var pathErr2 *os.PathError
		assert.ErrorAs(t, err1, &pathErr1, description+": wrong error type")
		assert.ErrorAs(t, err2, &pathErr2, description+": wrong error type")
		assert.Equal(t, pathErr1.Path, pathErr1.Path, description+": different file path")

		// on Linux validates that the syscall error is the same
		if runtime.GOOS == "linux" {
			var syscallErr1 syscall.Errno
			var syscallErr2 syscall.Errno
			assert.ErrorAs(t, err1, &syscallErr1, description+": wrong error type")
			assert.ErrorAs(t, err2, &syscallErr2, description+": wrong error type")
			assert.Equal(t, syscallErr1, syscallErr2, description+": same error for existing path (follow=false)")
		}
	}

	err = Lexists(tempDir)
	_, err2 := os.Lstat(tempDir)
	assertSameError(err, err2, "same error for existing path (follow=false)")

	err = Lexists("foo123shouldnotexist")
	_, err2 = os.Lstat("foo123shouldnotexist")
	assertSameError(err, err2, "same error for not existing path (follow=false)")

	err = Lexists(symlinkPath)
	_, err2 = os.Lstat(symlinkPath)
	assertSameError(err, err2, "same error for existing symlink (follow=false)")

	err = Exists(symlinkPath)
	_, err2 = os.Stat(symlinkPath)
	assertSameError(err, err2, "same error for existing symlink (follow=true)")

	err = Lexists(danglingSymlinkPath)
	_, err2 = os.Lstat(danglingSymlinkPath)
	assertSameError(err, err2, "same error for not existing symlink (follow=false)")

	err = Exists(danglingSymlinkPath)
	_, err2 = os.Stat(danglingSymlinkPath)
	assertSameError(err, err2, "same error for not existing symlink (follow=true)")
}

func BenchmarkExists(b *testing.B) {
	tempDir := b.TempDir()
	for i := 0; i < b.N; i++ {
		_ = Exists(tempDir)
		_ = Lexists(tempDir)
	}
}

func BenchmarkStat(b *testing.B) {
	tempDir := b.TempDir()
	for i := 0; i < b.N; i++ {
		_, _ = os.Stat(tempDir)
		_, _ = os.Lstat(tempDir)
	}
}
