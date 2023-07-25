package unionbackfill

import (
	"archive/tar"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/system"
	"github.com/containers/storage/pkg/tarbackfill"
	"github.com/stretchr/testify/require"
)

func TestBackfiller(t *testing.T) {
	tmp := t.TempDir()
	subdirs := make([]string, 0, 10)
	lower := filepath.Join(tmp, "lower")
	require.NoError(t, os.Mkdir(lower, 0o755))
	for i := 0; i < cap(subdirs); i++ {
		subdir := filepath.Join(lower, fmt.Sprintf("%d", i))
		require.NoError(t, os.Mkdir(subdir, 0o755))
		subdirs = append(subdirs, subdir)
	}
	epoch := time.Time{}.UTC()
	early := time.Unix(1000000000, 234567).UTC()
	// mark some parts of lowers as opaque (i.e., stop here when looking for content)
	for _, opaqueDir := range []string{
		"4/a/b/c/d",
		"5/a/b/c",
		"6/a/b",
		"7/a",
		"8",
		".",
	} {
		// create the opaque marker in the specified directory
		parent := filepath.Join(lower, opaqueDir)
		err := os.MkdirAll(parent, 0o755)
		require.NoError(t, err)
		f, err := os.Create(filepath.Join(parent, archive.WhiteoutOpaqueDir))
		require.NoError(t, err)
		f.Close()
		// create a piece of content that we should see in the opaque directory
		f, err = os.Create(filepath.Join(parent, "in-opaque"))
		require.NoError(t, err)
		f.Close()
		os.Chtimes(filepath.Join(parent, "in-opaque"), epoch, epoch)
	}
	// some content that should be hidden because it's below an opaque, higher directory
	for _, hiddenItemDir := range []string{
		"5/a/b/c/d",
		"6/a/b/c",
		"7/a/b",
		"8/a",
		"9",
	} {
		parent := filepath.Join(lower, hiddenItemDir)
		err := os.MkdirAll(parent, 0o755)
		require.NoError(t, err)
		f, err := os.Create(filepath.Join(parent, "hidden"))
		require.NoError(t, err)
		f.Close()
	}
	// some content that we expect to be able to find
	for _, visibleItemDir := range []string{
		"2/a/b/c/d/e",
		"3/a/b/c/d",
		"4/a/b/c",
		"5/a/b",
		"6/a",
	} {
		parent := filepath.Join(lower, visibleItemDir)
		err := os.MkdirAll(parent, 0o755)
		require.NoError(t, err)
		f, err := os.Create(filepath.Join(parent, "visible"))
		require.NoError(t, err)
		require.NoError(t, f.Chmod(0o640))
		f.Close()
		require.NoError(t, os.Chtimes(filepath.Join(parent, "visible"), early, early))
	}
	var backfiller tarbackfill.Backfiller = NewBackfiller(nil, subdirs)
	testCases := []struct {
		requested, actual string
	}{
		{"a/b/c/d/hidden", ""},
		{"a/b/c/hidden", ""},
		{"a/b/hidden", ""},
		{"a/hidden", ""},
		{"hidden", ""},
		{"a/b/c/d/in-opaque", "4/a/b/c/d/in-opaque"},
		{"a/b/c/in-opaque", "5/a/b/c/in-opaque"},
		{"a/b/in-opaque", "6/a/b/in-opaque"},
		{"a/in-opaque", "7/a/in-opaque"},
		{"in-opaque", "8/in-opaque"},
		{"a/b/c/d/e/visible", "2/a/b/c/d/e/visible"},
		{"a/b/c/d/visible", "3/a/b/c/d/visible"},
		{"a/b/c/visible", "4/a/b/c/visible"},
		{"a/b/visible", "5/a/b/visible"},
		{"a/visible", "6/a/visible"},
	}
	for testCase := range testCases {
		t.Run(testCases[testCase].requested, func(t *testing.T) {
			hdr, err := backfiller.Backfill(testCases[testCase].requested)
			require.NoError(t, err)
			if testCases[testCase].actual == "" {
				require.Nilf(t, hdr, "expected to not find content for path %q", testCases[testCase].requested)
			} else {
				require.NotNilf(t, hdr, "expected to find content for path %q", testCases[testCase].requested)
				info, err := os.Lstat(filepath.Join(lower, testCases[testCase].actual))
				require.NoErrorf(t, err, "internal error looking for %q", testCases[testCase].actual)
				expectedHdr, err := tar.FileInfoHeader(info, "")
				require.NoErrorf(t, err, "internal error converting info about %q to a header", testCases[testCase].actual)
				require.NotNilf(t, expectedHdr, "internal error converting info about %q to a header", testCases[testCase].actual)
				expectedHdr.Name = testCases[testCase].requested
				require.Equalf(t, *expectedHdr, *hdr, "unexpected header values for %q", testCases[testCase].actual)
			}
		})
	}
}

func TestRedirectBackfiller(t *testing.T) {
	tmp := t.TempDir()
	mergedDir := filepath.Join(tmp, "merged")
	require.NoError(t, os.Mkdir(mergedDir, 0o755))
	workDir := filepath.Join(tmp, "work")
	require.NoError(t, os.Mkdir(workDir, 0o755))

	directoryMode := 0o710
	directoryUid := 7
	directoryGid := 8
	now := time.Unix(time.Now().Unix(), 0)
	defaultMode := 0o755

	// create a directory we'll move around and put a directory under it and content in _that_
	layerDir := filepath.Join(tmp, "layer0")
	layerDirs := []string{layerDir}
	targetDir := filepath.Join(layerDirs[0], "a", "b", "c", "d", "e", "f", "g", "h", "template")
	require.NoError(t, os.MkdirAll(targetDir, fs.FileMode(defaultMode)))
	require.NoError(t, ioutil.WriteFile(filepath.Join(targetDir, "file"), []byte("some content"), 0o644))
	require.NoError(t, os.Chown(targetDir, directoryUid, directoryGid))
	require.NoError(t, os.Chmod(targetDir, fs.FileMode(directoryMode)))
	require.NoError(t, os.Chtimes(targetDir, now, now))

	// construct the location of the parent directory that we'll move once the overlay fs is mounted
	targetDir = strings.ReplaceAll(filepath.Dir(targetDir), layerDir, mergedDir)

	mount := func() error {
		redirectArg := "redirect_dir=on"
		workdirArg := fmt.Sprintf("workdir=%s", workDir)
		upperArg := fmt.Sprintf("upperdir=%s", layerDirs[0])
		var lowers []string
		for i := 1; i < len(layerDirs); i++ {
			lowers = append(lowers, layerDirs[i])
		}
		lowersArg := fmt.Sprintf("lowerdir=%s", strings.Join(lowers, ":"))
		mountOptArgs := []string{redirectArg, workdirArg, lowersArg, upperArg}
		mountOpts := strings.Join(mountOptArgs, ",")
		return syscall.Mount("none", mergedDir, "overlay", 0, mountOpts)
	}
	unmount := func() error {
		return syscall.Unmount(mergedDir, 0)
	}
	defer unmount()

	// mount, then rename the mobile directory through the overlay mount
	layerDir = filepath.Join(tmp, "layer1")
	layerDirs = append([]string{layerDir}, layerDirs...)
	require.NoError(t, os.Mkdir(layerDir, fs.FileMode(defaultMode)))
	require.NoError(t, mount())
	newTargetDir := filepath.Join(mergedDir, "a", "b", "c", "d", "e", "f", "g", "h-new")
	err := os.Rename(targetDir, newTargetDir)
	if err != nil && (errors.Is(err, syscall.EXDEV) || errors.Is(err, syscall.EINVAL)) {
		t.Skipf("unexpected error %v during rename - unable to test with redirect_dir=on", err)
	}
	require.NoError(t, err)
	targetDir = newTargetDir
	require.NoError(t, unmount())

	// check that the kernel attached a "redirect" overlay attribute to the
	// directory in the upper layer
	xval, err := system.Lgetxattr(strings.ReplaceAll(targetDir, mergedDir, layerDir), archive.GetOverlayXattrName("redirect"))
	if err != nil || len(xval) == 0 {
		t.Skipf("kernel did not set redirect attribute in upper directory, can't test this")
	}

	// add another layer in which we move it again
	layerDir = filepath.Join(tmp, "layer2")
	layerDirs = append([]string{layerDir}, layerDirs...)
	require.NoError(t, os.Mkdir(layerDir, fs.FileMode(defaultMode)))
	require.NoError(t, mount())
	newTargetDir = filepath.Join(mergedDir, "look-in-a-subdirectory")
	require.NoError(t, os.Rename(targetDir, newTargetDir))
	targetDir = newTargetDir
	require.NoError(t, unmount())

	// add another layer in which we move it again
	layerDir = filepath.Join(tmp, "layer3")
	layerDirs = append([]string{layerDir}, layerDirs...)
	require.NoError(t, os.Mkdir(layerDir, fs.FileMode(defaultMode)))
	require.NoError(t, mount())
	newTargetDir = filepath.Join(mergedDir, "a", "b", "c", "d", "look-in-a-parent-sibling-directory")
	require.NoError(t, os.Rename(targetDir, newTargetDir))
	targetDir = newTargetDir
	require.NoError(t, unmount())

	// add another layer in which we move it again
	layerDir = filepath.Join(tmp, "layer4")
	layerDirs = append([]string{layerDir}, layerDirs...)
	require.NoError(t, os.Mkdir(layerDir, fs.FileMode(defaultMode)))
	require.NoError(t, mount())
	newTargetDir = filepath.Join(mergedDir, "a", "b", "c", "d", "look-in-a-sibling-directory")
	require.NoError(t, os.Rename(targetDir, newTargetDir))
	targetDir = newTargetDir
	require.NoError(t, unmount())

	// add another layer in which we move it again
	layerDir = filepath.Join(tmp, "layer5")
	layerDirs = append([]string{layerDir}, layerDirs...)
	require.NoError(t, os.Mkdir(layerDir, fs.FileMode(defaultMode)))
	require.NoError(t, mount())
	require.NoError(t, os.Mkdir(filepath.Join(mergedDir, "a", "b", "c", "d", "e", "f", "g", "h"), 0o755))
	newTargetDir = filepath.Join(mergedDir, "a", "b", "c", "d", "e", "f", "g", "h", "template")
	require.NoError(t, os.Rename(targetDir, newTargetDir))
	// targetDir = newTargetDir
	require.NoError(t, unmount())

	// add another layer in which nothing happens
	layerDir = filepath.Join(tmp, "layer6")
	layerDirs = append([]string{layerDir}, layerDirs...)

	// start looking around
	backfiller := NewBackfiller(nil, layerDirs)
	hdr, err := backfiller.Backfill(path.Join("/", "a", "b", "c"))
	require.NoError(t, err)
	require.NotNil(t, hdr)
	require.Equal(t, path.Join("a", "b", "c")+"/", hdr.Name)
	hdr, err = backfiller.Backfill(path.Join("a", "b", "c"))
	require.NoError(t, err)
	require.NotNil(t, hdr)
	require.Equal(t, path.Join("a", "b", "c")+"/", hdr.Name)
	hdr, err = backfiller.Backfill(path.Join("a", "b", "d"))
	require.NoError(t, err)
	require.Nil(t, hdr)
	hdr, err = backfiller.Backfill(path.Join("a", "b", "c", "d", "e", "f", "g", "h", "template"))
	require.NoError(t, err)
	require.NotNil(t, hdr)
	require.Equal(t, int64(defaultMode), int64(hdr.Mode))
	require.Equal(t, 0, int(hdr.Uid))
	require.Equal(t, 0, int(hdr.Gid))
	hdr, err = backfiller.Backfill(path.Join("a", "b", "c", "d", "e", "f", "g", "h", "template", "template"))
	require.NoError(t, err)
	require.NotNil(t, hdr)
	require.Equal(t, int64(directoryMode), int64(hdr.Mode))
	require.Equal(t, directoryUid, int(hdr.Uid))
	require.Equal(t, directoryGid, int(hdr.Gid))
	hdr, err = backfiller.Backfill(path.Join("a", "b", "c", "d", "e", "f", "g", "h", "template", "template", "file"))
	require.NoError(t, err)
	require.NotNil(t, hdr)
	require.Equal(t, int64(0o644), int64(hdr.Mode))
	require.Equal(t, os.Getuid(), int(hdr.Uid))
	require.Equal(t, os.Getgid(), int(hdr.Gid))
}
