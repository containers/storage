//go:build linux

package copy

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/containers/storage/pkg/system"
	"golang.org/x/sys/unix"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCopy(t *testing.T) {
	copyWithFileRange := true
	copyWithFileClone := true
	doCopyTest(t, &copyWithFileRange, &copyWithFileClone)
}

func TestCopyWithoutRange(t *testing.T) {
	copyWithFileRange := false
	copyWithFileClone := false
	doCopyTest(t, &copyWithFileRange, &copyWithFileClone)
}

func TestCopyDir(t *testing.T) {
	srcDir := t.TempDir()
	populateSrcDir(t, srcDir, 3)

	dstDir := t.TempDir()

	assert.Check(t, DirCopy(srcDir, dstDir, Content, false))
	assert.NilError(t, filepath.Walk(srcDir, func(srcPath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Rebase path
		relPath, err := filepath.Rel(srcDir, srcPath)
		assert.NilError(t, err)
		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(dstDir, relPath)
		assert.NilError(t, err)

		// If we add non-regular dirs and files to the test
		// then we need to add more checks here.
		dstFileInfo, err := os.Lstat(dstPath)
		assert.NilError(t, err)

		srcFileSys := f.Sys().(*syscall.Stat_t)
		dstFileSys := dstFileInfo.Sys().(*syscall.Stat_t)

		t.Log(relPath)
		if srcFileSys.Dev == dstFileSys.Dev {
			assert.Check(t, srcFileSys.Ino != dstFileSys.Ino)
		}
		// Todo: check size, and ctim is not equal
		/// on filesystems that have granular ctimes
		assert.Check(t, is.DeepEqual(srcFileSys.Mode, dstFileSys.Mode))
		assert.Check(t, is.DeepEqual(srcFileSys.Uid, dstFileSys.Uid))
		assert.Check(t, is.DeepEqual(srcFileSys.Gid, dstFileSys.Gid))
		assert.Check(t, is.DeepEqual(srcFileSys.Mtim, dstFileSys.Mtim))

		return nil
	}))
}

func randomMode(baseMode int) os.FileMode {
	for i := range 7 {
		baseMode = baseMode | (1&rand.Intn(2))<<uint(i)
	}
	return os.FileMode(baseMode)
}

func populateSrcDir(t *testing.T, srcDir string, remainingDepth int) {
	if remainingDepth == 0 {
		socketPath := filepath.Join(srcDir, "srcsocket")
		s, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
		assert.NilError(t, err)
		s.SetUnlinkOnClose(false)
		s.Close()
		return
	}
	aTime := time.Unix(rand.Int63(), 0)
	mTime := time.Unix(rand.Int63(), 0)

	for i := range 10 {
		dirName := filepath.Join(srcDir, fmt.Sprintf("srcdir-%d", i))
		// Owner all bits set
		assert.NilError(t, os.Mkdir(dirName, randomMode(0o700)))
		populateSrcDir(t, dirName, remainingDepth-1)
		assert.NilError(t, system.Chtimes(dirName, aTime, mTime))
	}

	for i := range 10 {
		fileName := filepath.Join(srcDir, fmt.Sprintf("srcfile-%d", i))
		// Owner read bit set
		assert.NilError(t, os.WriteFile(fileName, []byte{}, randomMode(0o400)))
		assert.NilError(t, system.Chtimes(fileName, aTime, mTime))
	}
}

func doCopyTest(t *testing.T, copyWithFileRange, copyWithFileClone *bool) {
	dir := t.TempDir()
	srcFilename := filepath.Join(dir, "srcFilename")
	dstFilename := filepath.Join(dir, "dstilename")

	r := rand.New(rand.NewSource(0))
	buf := make([]byte, 1024)
	_, err := r.Read(buf)
	assert.NilError(t, err)
	assert.NilError(t, os.WriteFile(srcFilename, buf, 0o777))
	fileinfo, err := os.Stat(srcFilename)
	assert.NilError(t, err)

	assert.NilError(t, CopyRegular(srcFilename, dstFilename, fileinfo, copyWithFileRange, copyWithFileClone))
	readBuf, err := os.ReadFile(dstFilename)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(buf, readBuf))
}

func TestCopyHardlink(t *testing.T) {
	var srcFile1FileInfo, srcFile2FileInfo, dstFile1FileInfo, dstFile2FileInfo unix.Stat_t

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile1 := filepath.Join(srcDir, "file1")
	srcFile2 := filepath.Join(srcDir, "file2")
	dstFile1 := filepath.Join(dstDir, "file1")
	dstFile2 := filepath.Join(dstDir, "file2")
	assert.NilError(t, os.WriteFile(srcFile1, []byte{}, 0o777))
	assert.NilError(t, os.Link(srcFile1, srcFile2))

	assert.Check(t, DirCopy(srcDir, dstDir, Content, false))

	assert.NilError(t, unix.Stat(srcFile1, &srcFile1FileInfo))
	assert.NilError(t, unix.Stat(srcFile2, &srcFile2FileInfo))
	assert.Equal(t, srcFile1FileInfo.Ino, srcFile2FileInfo.Ino)

	assert.NilError(t, unix.Stat(dstFile1, &dstFile1FileInfo))
	assert.NilError(t, unix.Stat(dstFile2, &dstFile2FileInfo))
	assert.Check(t, is.Equal(dstFile1FileInfo.Ino, dstFile2FileInfo.Ino))
}
