package archive

import (
	"archive/tar"
	"io"
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
//
//	└── f1 # whiteout, 0000
func setupOverlayTestDir(t *testing.T, src string) {
	// Create opaque directory containing single file and permission 0700
	err := os.Mkdir(filepath.Join(src, "d1"), 0o700)
	require.NoError(t, err)

	err = system.Lsetxattr(filepath.Join(src, "d1"), getOverlayOpaqueXattrName(), []byte("y"), 0)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(src, "d1", "f1"), []byte{}, 0o600)
	require.NoError(t, err)

	// Create another opaque directory containing single file but with permission 0750
	err = os.Mkdir(filepath.Join(src, "d2"), 0o750)
	require.NoError(t, err)

	err = system.Lsetxattr(filepath.Join(src, "d2"), getOverlayOpaqueXattrName(), []byte("y"), 0)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(src, "d2", "f1"), []byte{}, 0o660)
	require.NoError(t, err)

	// Create regular directory with deleted file
	err = os.Mkdir(filepath.Join(src, "d3"), 0o700)
	require.NoError(t, err)

	err = system.Mknod(filepath.Join(src, "d3", "f1"), unix.S_IFCHR, 0)
	require.NoError(t, err)
}

func setupOverlayLowerDir(t *testing.T, lower string) {
	// Create a subdirectory to use as the "lower layer"'s copy of a deleted directory
	err := os.Mkdir(filepath.Join(lower, "d1"), 0o700)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(lower, "d1", "f1"), []byte{}, 0o600)
	require.NoError(t, err)
}

func checkOpaqueness(t *testing.T, path string, opaque string) {
	xattrOpaque, err := system.Lgetxattr(path, getOverlayOpaqueXattrName())
	require.NoError(t, err)

	if string(xattrOpaque) != opaque {
		t.Fatalf("Unexpected opaque value: %q, expected %q", string(xattrOpaque), opaque)
	}
}

func checkOverlayWhiteout(t *testing.T, path string) {
	stat, err := os.Stat(path)
	require.NoError(t, err)

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
	require.NoError(t, err)

	if stat.Mode() != perm {
		t.Fatalf("Unexpected file mode for %s: %o, expected %o", path, stat.Mode(), perm)
	}
}

func TestOverlayTarUntar(t *testing.T) {
	oldMask, err := system.Umask(0)
	require.NoError(t, err)
	defer func() {
		_, _ = system.Umask(oldMask) // Ignore err. This can only fail with ErrNotSupportedPlatform, in which case we would have failed above.
	}()

	src := t.TempDir()
	setupOverlayTestDir(t, src)

	lower := t.TempDir()
	setupOverlayLowerDir(t, lower)

	dst := t.TempDir()

	options := &TarOptions{
		Compression:    Uncompressed,
		WhiteoutFormat: OverlayWhiteoutFormat,
		WhiteoutData:   []string{lower},
	}
	archive, err := TarWithOptions(src, options)
	require.NoError(t, err)
	defer archive.Close()

	err = Untar(archive, dst, options)
	require.NoError(t, err)

	checkFileMode(t, filepath.Join(dst, "d1"), 0o700|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d2"), 0o750|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d3"), 0o700|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d1", "f1"), 0o600)
	checkFileMode(t, filepath.Join(dst, "d2", "f1"), 0o660)
	checkFileMode(t, filepath.Join(dst, "d3", "f1"), os.ModeCharDevice|os.ModeDevice)

	checkOpaqueness(t, filepath.Join(dst, "d1"), "y")
	checkOpaqueness(t, filepath.Join(dst, "d2"), "")
	checkOpaqueness(t, filepath.Join(dst, "d3"), "")
	checkOverlayWhiteout(t, filepath.Join(dst, "d3", "f1"))
}

func TestOverlayTarAUFSUntar(t *testing.T) {
	oldMask, err := system.Umask(0)
	require.NoError(t, err)
	defer func() {
		_, _ = system.Umask(oldMask) // Ignore err. This can only fail with ErrNotSupportedPlatform, in which case we would have failed above.
	}()

	src := t.TempDir()
	setupOverlayTestDir(t, src)

	lower := t.TempDir()
	setupOverlayLowerDir(t, lower)

	dst := t.TempDir()

	archive, err := TarWithOptions(src, &TarOptions{
		Compression:    Uncompressed,
		WhiteoutFormat: OverlayWhiteoutFormat,
		WhiteoutData:   []string{lower},
	})
	require.NoError(t, err)
	defer archive.Close()

	err = Untar(archive, dst, &TarOptions{
		Compression:    Uncompressed,
		WhiteoutFormat: AUFSWhiteoutFormat,
	})
	require.NoError(t, err)

	checkFileMode(t, filepath.Join(dst, "d1"), 0o700|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d1", WhiteoutOpaqueDir), 0o700)
	checkFileMode(t, filepath.Join(dst, "d2"), 0o750|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d3"), 0o700|os.ModeDir)
	checkFileMode(t, filepath.Join(dst, "d1", "f1"), 0o600)
	checkFileMode(t, filepath.Join(dst, "d2", "f1"), 0o660)
	checkFileMode(t, filepath.Join(dst, "d3", WhiteoutPrefix+"f1"), 0)
}

func TestNestedOverlayWhiteouts(t *testing.T) {
	reader, writer := io.Pipe()

	go func() {
		tw := tar.NewWriter(writer)
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     ".wh.foo",
			Size:     0,
			Uid:      os.Geteuid(),
			Gid:      os.Getegid(),
		}))
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     "foo/.wh.bar",
			Size:     0,
			Uid:      os.Geteuid(),
			Gid:      os.Getegid(),
		}))
		require.NoError(t, tw.Close())
	}()

	dst := t.TempDir()

	err := Untar(reader, dst, &TarOptions{
		Compression:    Uncompressed,
		WhiteoutFormat: OverlayWhiteoutFormat,
	})
	require.NoError(t, err)
	checkFileMode(t, filepath.Join(dst, "foo"), os.ModeDevice|os.ModeCharDevice)
}
