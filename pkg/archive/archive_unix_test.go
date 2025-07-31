//go:build !windows

package archive

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"

	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestCanonicalTarNameForPath(t *testing.T) {
	cases := []struct{ in, expected string }{
		{"foo", "foo"},
		{"foo/bar", "foo/bar"},
		{"foo/dir/", "foo/dir/"},
	}
	for _, v := range cases {
		if out, err := CanonicalTarNameForPath(v.in); err != nil {
			t.Fatalf("cannot get canonical name for path: %s: %v", v.in, err)
		} else if out != v.expected {
			t.Fatalf("wrong canonical tar name. expected:%s got:%s", v.expected, out)
		}
	}
}

func TestCanonicalTarName(t *testing.T) {
	cases := []struct {
		in       string
		isDir    bool
		expected string
	}{
		{"foo", false, "foo"},
		{"foo", true, "foo/"},
		{"foo/bar", false, "foo/bar"},
		{"foo/bar", true, "foo/bar/"},
	}
	for _, v := range cases {
		if out, err := canonicalTarName(v.in, v.isDir); err != nil {
			t.Fatalf("cannot get canonical name for path: %s: %v", v.in, err)
		} else if out != v.expected {
			t.Fatalf("wrong canonical tar name. expected:%s got:%s", v.expected, out)
		}
	}
}

func TestChmodTarEntry(t *testing.T) {
	cases := []struct {
		in, expected os.FileMode
	}{
		{0o000, 0o000},
		{0o777, 0o777},
		{0o644, 0o644},
		{0o755, 0o755},
		{0o444, 0o444},
	}
	for _, v := range cases {
		if out := chmodTarEntry(v.in); out != v.expected {
			t.Fatalf("wrong chmod. expected:%v got:%v", v.expected, out)
		}
	}
}

func TestTarWithHardLink(t *testing.T) {
	origin := t.TempDir()

	err := os.WriteFile(filepath.Join(origin, "1"), []byte("hello world"), 0o700)
	require.NoError(t, err)

	for i := 2; i <= 10; i++ {
		err = os.Link(filepath.Join(origin, "1"), filepath.Join(origin, strconv.Itoa(i)))
		require.NoError(t, err)
	}

	var i1, i2 uint64
	i1, err = getNlink(filepath.Join(origin, "1"))
	require.NoError(t, err)

	// sanity check that we can hardlink
	if i1 != 10 {
		t.Skipf("skipping since hardlinks don't work here; expected 10 links, got %d", i1)
	}

	dest := t.TempDir()

	// we'll do this in two steps to separate failure
	fh, err := Tar(origin, Uncompressed)
	require.NoError(t, err)

	// ensure we can read the whole thing with no error, before writing back out
	buf, err := io.ReadAll(fh)
	require.NoError(t, err)

	bRdr := bytes.NewReader(buf)
	err = Untar(bRdr, dest, &TarOptions{Compression: Uncompressed})
	require.NoError(t, err)

	i1, err = getInode(filepath.Join(dest, "1"))
	require.NoError(t, err)

	i2, err = getInode(filepath.Join(dest, "2"))
	require.NoError(t, err)

	assert.Equal(t, i1, i2)

	// check that hard link entries aren't listing hard link entries as their targets
	headers, err := gatherHeaders(bytes.NewReader(buf))
	require.NoError(t, err)
	for _, hdr := range headers {
		if hdr.Typeflag == tar.TypeLink {
			target := headers[hdr.Linkname]
			require.NotNilf(t, target, "entry for link target %q", hdr.Linkname)
			require.NotEqualValues(t, tar.TypeLink, target.Typeflag, "link target should not have been another link")
		}
	}
}

func TestTarWithHardLinkAndRebase(t *testing.T) {
	tmpDir := t.TempDir()

	origin := filepath.Join(tmpDir, "origin")
	err := os.Mkdir(origin, 0o700)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(origin, "1"), []byte("hello world"), 0o700)
	require.NoError(t, err)

	err = os.Link(filepath.Join(origin, "1"), filepath.Join(origin, "2"))
	require.NoError(t, err)

	var i1, i2 uint64
	i1, err = getNlink(filepath.Join(origin, "1"))
	require.NoError(t, err)

	// sanity check that we can hardlink
	if i1 != 2 {
		t.Skipf("skipping since hardlinks don't work here; expected 2 links, got %d", i1)
	}

	dest := filepath.Join(tmpDir, "dest")
	bRdr, err := TarResourceRebase(origin, "origin")
	require.NoError(t, err)

	dstDir, srcBase := SplitPathDirEntry(origin)
	_, dstBase := SplitPathDirEntry(dest)
	content := RebaseArchiveEntries(bRdr, srcBase, dstBase)
	err = Untar(content, dstDir, &TarOptions{Compression: Uncompressed, NoLchown: true, NoOverwriteDirNonDir: true})
	require.NoError(t, err)

	i1, err = getInode(filepath.Join(dest, "1"))
	require.NoError(t, err)
	i2, err = getInode(filepath.Join(dest, "2"))
	require.NoError(t, err)

	assert.Equal(t, i1, i2)
}

func gatherHeaders(reader io.Reader) (map[string]*tar.Header, error) {
	headers := make(map[string]*tar.Header)
	tr := tar.NewReader(reader)
	hdr, err := tr.Next()
	for hdr != nil {
		header := *hdr
		headers[hdr.Name] = &header
		if err != nil {
			break
		}
		hdr, err = tr.Next()
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return headers, err
}

func getNlink(path string) (uint64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	statT, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("expected type *syscall.Stat_t, got %t", stat.Sys())
	}
	return uint64(statT.Nlink), nil //nolint:unconvert // Need the conversion for e.g. linux/arm64.
}

func getInode(path string) (uint64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	statT, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("expected type *syscall.Stat_t, got %t", stat.Sys())
	}
	return statT.Ino, nil
}

func TestTarWithBlockCharFifo(t *testing.T) {
	origin := t.TempDir()

	err := os.WriteFile(filepath.Join(origin, "1"), []byte("hello world"), 0o700)
	require.NoError(t, err)

	err = system.Mknod(filepath.Join(origin, "2"), unix.S_IFBLK, system.Mkdev(int64(12), int64(5)))
	require.NoError(t, err)
	err = system.Mknod(filepath.Join(origin, "3"), unix.S_IFCHR, system.Mkdev(int64(12), int64(5)))
	require.NoError(t, err)
	if runtime.GOOS != freebsd {
		// On FreeBSD mknod with S_IFIFO requires the dev argument to be zero.
		err = system.Mknod(filepath.Join(origin, "4"), unix.S_IFIFO, system.Mkdev(int64(12), int64(5)))
	}
	require.NoError(t, err)

	dest := t.TempDir()

	// we'll do this in two steps to separate failure
	fh, err := Tar(origin, Uncompressed)
	require.NoError(t, err)

	// ensure we can read the whole thing with no error, before writing back out
	buf, err := io.ReadAll(fh)
	require.NoError(t, err)

	bRdr := bytes.NewReader(buf)
	err = Untar(bRdr, dest, &TarOptions{Compression: Uncompressed})
	require.NoError(t, err)

	changes, err := ChangesDirs(origin, &idtools.IDMappings{}, dest, &idtools.IDMappings{})
	require.NoError(t, err)

	if len(changes) > 0 {
		t.Fatalf("Tar with special device (block, char, fifo) should keep them (recreate them when untar) : %v", changes)
	}
}

// TestTarUntarWithXattr is Unix as Lsetxattr is not supported on Windows
func TestTarUntarWithXattr(t *testing.T) {
	if runtime.GOOS == solaris || runtime.GOOS == freebsd {
		t.Skip()
	}
	origin := t.TempDir()
	err := os.WriteFile(filepath.Join(origin, "1"), []byte("hello world"), 0o700)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(origin, "2"), []byte("welcome!"), 0o700)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(origin, "3"), []byte("will be ignored"), 0o700)
	require.NoError(t, err)
	encoded := [20]byte{0, 0, 0, 2}
	err = system.Lsetxattr(filepath.Join(origin, "2"), "security.capability", encoded[:], 0)
	require.NoError(t, err)
	err = system.Lsetxattr(filepath.Join(origin, "1"), "user.test", []byte("helloWord"), 0)
	require.NoError(t, err)

	for _, c := range []Compression{
		Uncompressed,
		Gzip,
	} {
		changes, err := tarUntar(t, origin, &TarOptions{
			Compression:     c,
			ExcludePatterns: []string{"3"},
		})
		if err != nil {
			t.Fatalf("Error tar/untar for compression %s: %s", c.Extension(), err)
		}

		if len(changes) != 1 || changes[0].Path != "/3" {
			t.Fatalf("Unexpected differences after tarUntar: %v", changes)
		}

		capability, err := system.Lgetxattr(filepath.Join(origin, "2"), "security.capability")
		require.NoError(t, err)
		assert.Equal(t, encoded[:], capability)

		test, err := system.Lgetxattr(filepath.Join(origin, "1"), "user.test")
		require.NoError(t, err)
		assert.Equal(t, []byte("helloWord"), test)
	}
}
