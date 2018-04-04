package archive

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/containers/storage/pkg/system"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// setupOverlayTestDir creates files in a directory with overlay whiteouts
// Tree layout
// .
// ├── d1     # opaque, 0700
// │   └── f1 # empty file, 0600
// ├── d2     # opaque, 0750
// │   └── f1 # empty file, 0660
// └── d3     # 0700
//     └── f1 # whiteout, 0644
func setupOverlayTestDir(t *testing.T, src string) {
	// Create opaque directory containing single file and permission 0700
	err := os.Mkdir(filepath.Join(src, "d1"), 0700)
	assert.NilError(t, err)

	err = system.Lsetxattr(filepath.Join(src, "d1"), "trusted.overlay.opaque", []byte("y"), 0)
	assert.NilError(t, err)

	err = ioutil.WriteFile(filepath.Join(src, "d1", "f1"), []byte{}, 0600)
	assert.NilError(t, err)

	// Create another opaque directory containing single file but with permission 0750
	err = os.Mkdir(filepath.Join(src, "d2"), 0750)
	assert.NilError(t, err)

	err = system.Lsetxattr(filepath.Join(src, "d2"), "trusted.overlay.opaque", []byte("y"), 0)
	assert.NilError(t, err)

	err = ioutil.WriteFile(filepath.Join(src, "d2", "f1"), []byte{}, 0660)
	assert.NilError(t, err)

	// Create regular directory with deleted file
	err = os.Mkdir(filepath.Join(src, "d3"), 0700)
	assert.NilError(t, err)

	err = system.Mknod(filepath.Join(src, "d3", "f1"), unix.S_IFCHR, 0)
	assert.NilError(t, err)
}

func setupOverlayLowerDir(t *testing.T, lower string) {
	// Create a subdirectory to use as the "lower layer"'s copy of a deleted directory
	err := os.Mkdir(filepath.Join(lower, "d1"), 0700)
	assert.NilError(t, err)

	err = ioutil.WriteFile(filepath.Join(lower, "d1", "f1"), []byte{}, 0600)
	assert.NilError(t, err)
}

func checkOpaqueness(t *testing.T, path string, opaque string) {
	xattrOpaque, err := system.Lgetxattr(path, "trusted.overlay.opaque")
	assert.NilError(t, err)

	if string(xattrOpaque) != opaque {
		t.Fatalf("Unexpected opaque value: %q, expected %q", string(xattrOpaque), opaque)
	}

}

func checkOverlayWhiteout(t *testing.T, path string) {
	stat, err := os.Stat(path)
	assert.NilError(t, err)

	statT, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("Unexpected type: %t, expected *syscall.Stat_t", stat.Sys())
	}
	if statT.Rdev != 0 {
		t.Fatalf("Non-zero device number for whiteout")
	}
}

func checkFileMode(t *testing.T, path string, perm os.FileMode) {
	stat, err := os.Stat(path)
	assert.NilError(t, err)

	if stat.Mode() != perm {
		t.Fatalf("Unexpected file mode for %s: %o, expected %o", path, stat.Mode(), perm)
	}
}

func TestOverlayTarUntar(t *testing.T) {
	oldmask, err := system.Umask(0)
	assert.NilError(t, err)
	defer system.Umask(oldmask)

	src, err := ioutil.TempDir("", "storage-test-overlay-tar-src")
	assert.NilError(t, err)
	defer os.RemoveAll(src)

	setupOverlayTestDir(t, src)

	lower, err := ioutil.TempDir("", "storage-test-overlay-tar-lower")
	assert.NilError(t, err)
	defer os.RemoveAll(lower)

	setupOverlayLowerDir(t, lower)

	dst, err := ioutil.TempDir("", "storage-test-overlay-tar-dst")
	assert.NilError(t, err)
	defer os.RemoveAll(dst)

	options := &TarOptions{
		Compression:    Uncompressed,
		WhiteoutFormat: OverlayWhiteoutFormat,
		WhiteoutData:   []string{lower},
	}
	archive, err := TarWithOptions(src, options)
	assert.NilError(t, err)
	defer archive.Close()

	err = Untar(archive, dst, options)
	assert.NilError(t, err)

	checkFileMode(t, filepath.Join(dst, "d1"), 0700|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d2"), 0750|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d3"), 0700|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d1", "f1"), 0600)
	checkFileMode(t, filepath.Join(dst, "d2", "f1"), 0660)
	checkFileMode(t, filepath.Join(dst, "d3", "f1"), os.ModeCharDevice|os.ModeDevice)

	checkOpaqueness(t, filepath.Join(dst, "d1"), "y")
	checkOpaqueness(t, filepath.Join(dst, "d2"), "")
	checkOpaqueness(t, filepath.Join(dst, "d3"), "")
	checkOverlayWhiteout(t, filepath.Join(dst, "d3", "f1"))
}

func TestOverlayTarAUFSUntar(t *testing.T) {
	oldmask, err := system.Umask(0)
	assert.NilError(t, err)
	defer system.Umask(oldmask)

	src, err := ioutil.TempDir("", "storage-test-overlay-tar-src")
	assert.NilError(t, err)
	defer os.RemoveAll(src)

	setupOverlayTestDir(t, src)

	lower, err := ioutil.TempDir("", "storage-test-overlay-tar-lower")
	assert.NilError(t, err)
	defer os.RemoveAll(lower)

	setupOverlayLowerDir(t, lower)

	dst, err := ioutil.TempDir("", "storage-test-overlay-tar-dst")
	assert.NilError(t, err)
	defer os.RemoveAll(dst)

	archive, err := TarWithOptions(src, &TarOptions{
		Compression:    Uncompressed,
		WhiteoutFormat: OverlayWhiteoutFormat,
		WhiteoutData:   []string{lower},
	})
	assert.NilError(t, err)
	defer archive.Close()

	err = Untar(archive, dst, &TarOptions{
		Compression:    Uncompressed,
		WhiteoutFormat: AUFSWhiteoutFormat,
	})
	assert.NilError(t, err)

	checkFileMode(t, filepath.Join(dst, "d1"), 0700|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d1", WhiteoutOpaqueDir), 0700)
	checkFileMode(t, filepath.Join(dst, "d2"), 0750|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d3"), 0700|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d1", "f1"), 0600)
	checkFileMode(t, filepath.Join(dst, "d2", "f1"), 0660)
	checkFileMode(t, filepath.Join(dst, "d3", WhiteoutPrefix+"f1"), 0600)
}
