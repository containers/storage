package archive

import (
	"os"
	"os/exec"
	"path"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/system"
	"github.com/stretchr/testify/require"
)

func copyDir(src, dst string) error {
	cmd := exec.Command("cp", "-a", src, dst)
	if runtime.GOOS == solaris {
		cmd = exec.Command("gcp", "-a", src, dst)
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

type FileType uint32

const (
	Regular FileType = iota
	Dir
	Symlink
)

type FileData struct {
	filetype    FileType
	path        string
	contents    string
	permissions os.FileMode
}

func createSampleDir(t *testing.T, root string) {
	files := []FileData{
		{Regular, "file1", "file1\n", 0o600},
		{Regular, "file2", "file2\n", 0o666},
		{Regular, "file3", "file3\n", 0o404},
		{Regular, "file4", "file4\n", 0o600},
		{Regular, "file5", "file5\n", 0o600},
		{Regular, "file6", "file6\n", 0o600},
		{Regular, "file7", "file7\n", 0o600},
		{Dir, "dir1", "", 0o740},
		{Regular, "dir1/file1-1", "file1-1\n", 0o1444},
		{Regular, "dir1/file1-2", "file1-2\n", 0o666},
		{Dir, "dir2", "", 0o700},
		{Regular, "dir2/file2-1", "file2-1\n", 0o666},
		{Regular, "dir2/file2-2", "file2-2\n", 0o666},
		{Dir, "dir3", "", 0o700},
		{Regular, "dir3/file3-1", "file3-1\n", 0o666},
		{Regular, "dir3/file3-2", "file3-2\n", 0o666},
		{Dir, "dir4", "", 0o700},
		{Regular, "dir4/file3-1", "file4-1\n", 0o666},
		{Regular, "dir4/file3-2", "file4-2\n", 0o666},
		{Symlink, "symlink1", "target1", 0o666},
		{Symlink, "symlink2", "target2", 0o666},
		{Symlink, "symlink3", root + "/file1", 0o666},
		{Symlink, "symlink4", root + "/symlink3", 0o666},
		{Symlink, "dirSymlink", root + "/dir1", 0o740},
	}

	now := time.Now()
	for _, info := range files {
		p := path.Join(root, info.path)
		switch info.filetype {
		case Dir:
			err := os.MkdirAll(p, info.permissions)
			require.NoError(t, err)
		case Regular:
			err := os.WriteFile(p, []byte(info.contents), info.permissions)
			require.NoError(t, err)
		case Symlink:
			err := os.Symlink(info.contents, p)
			require.NoError(t, err)

			err = resetSymlinkTimes(p)
			require.NoError(t, err)
		}

		if info.filetype != Symlink {
			// Set a consistent ctime, atime for all files and dirs
			err := system.Chtimes(p, now, now)
			require.NoError(t, err)
		}
	}
}

func TestChangeString(t *testing.T) {
	modifyChange := Change{"change", ChangeModify}
	toString := modifyChange.String()
	if toString != "C change" {
		t.Fatalf("String() of a change with ChangeModify Kind should have been %s but was %s", "C change", toString)
	}
	addChange := Change{"change", ChangeAdd}
	toString = addChange.String()
	if toString != "A change" {
		t.Fatalf("String() of a change with ChangeAdd Kind should have been %s but was %s", "A change", toString)
	}
	deleteChange := Change{"change", ChangeDelete}
	toString = deleteChange.String()
	if toString != "D change" {
		t.Fatalf("String() of a change with ChangeDelete Kind should have been %s but was %s", "D change", toString)
	}
}

func TestChangesWithNoChanges(t *testing.T) {
	// TODO Windows. There may be a way of running this, but turning off for now
	// as createSampleDir uses symlinks.
	if runtime.GOOS == windows {
		t.Skip("symlinks on Windows")
	}
	rwLayer := t.TempDir()
	layer := t.TempDir()
	createSampleDir(t, layer)
	changes, err := Changes([]string{layer}, rwLayer)
	require.NoError(t, err)
	if len(changes) != 0 {
		t.Fatalf("Changes with no difference should have detect no changes, but detected %d", len(changes))
	}
}

func TestChangesWithChanges(t *testing.T) {
	// TODO Windows. There may be a way of running this, but turning off for now
	// as createSampleDir uses symlinks.
	if runtime.GOOS == windows {
		t.Skip("symlinks on Windows")
	}
	// Mock the readonly layer
	layer := t.TempDir()
	createSampleDir(t, layer)
	err := os.MkdirAll(path.Join(layer, "dir1/subfolder"), 0o740)
	require.NoError(t, err)

	// Mock the RW layer
	rwLayer := t.TempDir()

	// Create a folder in RW layer
	dir1 := path.Join(rwLayer, "dir1")
	err = os.MkdirAll(dir1, 0o740)
	require.NoError(t, err)
	deletedFile := path.Join(dir1, ".wh.file1-2")
	err = os.WriteFile(deletedFile, []byte{}, 0o600)
	require.NoError(t, err)
	modifiedFile := path.Join(dir1, "file1-1")
	err = os.WriteFile(modifiedFile, []byte{0x00}, 0o1444)
	require.NoError(t, err)
	// Let's add a subfolder for a newFile
	subfolder := path.Join(dir1, "subfolder")
	err = os.MkdirAll(subfolder, 0o740)
	require.NoError(t, err)
	newFile := path.Join(subfolder, "newFile")
	err = os.WriteFile(newFile, []byte{}, 0o740)
	require.NoError(t, err)

	changes, err := Changes([]string{layer}, rwLayer)
	require.NoError(t, err)

	expectedChanges := []Change{
		{"/dir1", ChangeModify},
		{"/dir1/file1-1", ChangeModify},
		{"/dir1/file1-2", ChangeDelete},
		{"/dir1/subfolder", ChangeModify},
		{"/dir1/subfolder/newFile", ChangeAdd},
	}
	checkChanges(t, expectedChanges, changes)
}

// See https://github.com/docker/docker/pull/13590
func TestChangesWithChangesGH13590(t *testing.T) {
	// TODO Windows. There may be a way of running this, but turning off for now
	// as createSampleDir uses symlinks.
	if runtime.GOOS == windows {
		t.Skip("symlinks on Windows")
	}
	baseLayer := t.TempDir()

	dir3 := path.Join(baseLayer, "dir1/dir2/dir3")
	err := os.MkdirAll(dir3, 0o7400)
	require.NoError(t, err)

	file := path.Join(dir3, "file.txt")
	err = os.WriteFile(file, []byte("hello"), 0o666)
	require.NoError(t, err)

	layer := t.TempDir()

	// Test creating a new file
	if err := copyDir(baseLayer+"/dir1", layer+"/"); err != nil {
		t.Fatalf("Cmd failed: %q", err)
	}

	os.Remove(path.Join(layer, "dir1/dir2/dir3/file.txt"))
	file = path.Join(layer, "dir1/dir2/dir3/file1.txt")
	err = os.WriteFile(file, []byte("bye"), 0o666)
	require.NoError(t, err)

	changes, err := Changes([]string{baseLayer}, layer)
	require.NoError(t, err)

	expectedChanges := []Change{
		{"/dir1", ChangeModify},
		{"/dir1/dir2", ChangeModify},
		{"/dir1/dir2/dir3", ChangeModify},
		{"/dir1/dir2/dir3/file1.txt", ChangeAdd},
	}
	checkChanges(t, expectedChanges, changes)

	// Now test changing a file
	layer = t.TempDir()

	if err := copyDir(baseLayer+"/dir1", layer+"/"); err != nil {
		t.Fatalf("Cmd failed: %q", err)
	}

	file = path.Join(layer, "dir1/dir2/dir3/file.txt")
	err = os.WriteFile(file, []byte("bye"), 0o666)
	require.NoError(t, err)

	changes, err = Changes([]string{baseLayer}, layer)
	require.NoError(t, err)

	expectedChanges = []Change{
		{"/dir1/dir2/dir3/file.txt", ChangeModify},
	}
	checkChanges(t, expectedChanges, changes)
}

// Create a directory, copy it, make sure we report no changes between the two
func TestChangesDirsEmpty(t *testing.T) {
	// TODO Windows. There may be a way of running this, but turning off for now
	// as createSampleDir uses symlinks.
	// TODO Should work for Solaris
	if runtime.GOOS == windows || runtime.GOOS == solaris {
		t.Skip("symlinks on Windows; gcp failure on Solaris")
	}
	src := t.TempDir()
	createSampleDir(t, src)
	dst := src + "-copy"
	err := copyDir(src, dst)
	require.NoError(t, err)
	changes, err := ChangesDirs(dst, &idtools.IDMappings{}, src, &idtools.IDMappings{})
	require.NoError(t, err)

	if len(changes) != 0 {
		t.Fatalf("Reported changes for identical dirs: %v", changes)
	}
}

func mutateSampleDir(t *testing.T, root string) {
	// Remove a regular file
	err := os.RemoveAll(path.Join(root, "file1"))
	require.NoError(t, err)

	// Remove a directory
	err = os.RemoveAll(path.Join(root, "dir1"))
	require.NoError(t, err)

	// Remove a symlink
	err = os.RemoveAll(path.Join(root, "symlink1"))
	require.NoError(t, err)

	// Rewrite a file
	err = os.WriteFile(path.Join(root, "file2"), []byte("fileNN\n"), 0o777)
	require.NoError(t, err)

	// Replace a file
	err = os.RemoveAll(path.Join(root, "file3"))
	require.NoError(t, err)
	err = os.WriteFile(path.Join(root, "file3"), []byte("fileMM\n"), 0o404)
	require.NoError(t, err)

	// Touch file
	err = system.Chtimes(path.Join(root, "file4"), time.Now().Add(time.Second), time.Now().Add(time.Second))
	require.NoError(t, err)

	// Replace file with dir
	err = os.RemoveAll(path.Join(root, "file5"))
	require.NoError(t, err)
	err = os.MkdirAll(path.Join(root, "file5"), 0o666)
	require.NoError(t, err)

	// Create new file
	err = os.WriteFile(path.Join(root, "filenew"), []byte("filenew\n"), 0o777)
	require.NoError(t, err)

	// Create new dir
	err = os.MkdirAll(path.Join(root, "dirnew"), 0o766)
	require.NoError(t, err)

	// Create a new symlink
	err = os.Symlink("targetnew", path.Join(root, "symlinknew"))
	require.NoError(t, err)

	// Change a symlink target, but keep same times and size
	symlink2 := path.Join(root, "symlink2")
	err = os.RemoveAll(symlink2)
	require.NoError(t, err)
	err = os.Symlink("target3", symlink2)
	require.NoError(t, err)
	err = resetSymlinkTimes(symlink2)
	require.NoError(t, err)

	// Replace dir with file
	err = os.RemoveAll(path.Join(root, "dir2"))
	require.NoError(t, err)
	err = os.WriteFile(path.Join(root, "dir2"), []byte("dir2\n"), 0o777)
	require.NoError(t, err)

	// Touch dir
	err = system.Chtimes(path.Join(root, "dir3"), time.Now().Add(time.Second), time.Now().Add(time.Second))
	require.NoError(t, err)
}

func TestChangesDirsMutated(t *testing.T) {
	// TODO Windows. There may be a way of running this, but turning off for now
	// as createSampleDir uses symlinks.
	// TODO Should work for Solaris
	if runtime.GOOS == windows || runtime.GOOS == solaris {
		t.Skip("symlinks on Windows; gcp failures on Solaris")
	}
	src := t.TempDir()
	createSampleDir(t, src)
	dst := src + "-copy"
	err := copyDir(src, dst)
	require.NoError(t, err)

	mutateSampleDir(t, dst)

	changes, err := ChangesDirs(dst, &idtools.IDMappings{}, src, &idtools.IDMappings{})
	require.NoError(t, err)

	sort.Sort(changesByPath(changes))

	expectedChanges := []Change{
		{"/dir1", ChangeDelete},
		{"/dir2", ChangeModify},
		{"/dir3", ChangeModify},
		{"/dirnew", ChangeAdd},
		{"/file1", ChangeDelete},
		{"/file2", ChangeModify},
		{"/file3", ChangeModify},
		{"/file4", ChangeModify},
		{"/file5", ChangeModify},
		{"/filenew", ChangeAdd},
		{"/symlink1", ChangeDelete},
		{"/symlink2", ChangeModify},
		{"/symlinknew", ChangeAdd},
	}

	for i := range max(len(changes), len(expectedChanges)) {
		if i >= len(expectedChanges) {
			t.Fatalf("unexpected change %s\n", changes[i].String())
		}
		if i >= len(changes) {
			t.Fatalf("no change for expected change %s\n", expectedChanges[i].String())
		}
		if changes[i].Path == expectedChanges[i].Path {
			if changes[i] != expectedChanges[i] {
				t.Fatalf("Wrong change for %s, expected %s, got %s\n", changes[i].Path, changes[i].String(), expectedChanges[i].String())
			}
		} else if changes[i].Path < expectedChanges[i].Path {
			t.Fatalf("unexpected change %s\n", changes[i].String())
		} else {
			t.Fatalf("no change for expected change %s != %s\n", expectedChanges[i].String(), changes[i].String())
		}
	}
}

func TestApplyLayer(t *testing.T) {
	// TODO Windows. There may be a way of running this, but turning off for now
	// as createSampleDir uses symlinks.
	// TODO Should work for Solaris
	if runtime.GOOS == windows || runtime.GOOS == solaris {
		t.Skip("symlinks on Windows; gcp failures on Solaris")
	}
	src := t.TempDir()
	createSampleDir(t, src)
	dst := src + "-copy"
	err := copyDir(src, dst)
	require.NoError(t, err)
	mutateSampleDir(t, dst)

	changes, err := ChangesDirs(dst, &idtools.IDMappings{}, src, &idtools.IDMappings{})
	require.NoError(t, err)

	layer, err := ExportChanges(dst, changes, nil, nil)
	require.NoError(t, err)

	layerCopy, err := NewTempArchive(layer, "")
	require.NoError(t, err)

	_, err = ApplyLayer(src, layerCopy)
	require.NoError(t, err)

	changes2, err := ChangesDirs(src, &idtools.IDMappings{}, dst, &idtools.IDMappings{})
	require.NoError(t, err)

	if len(changes2) != 0 {
		t.Fatalf("Unexpected differences after reapplying mutation: %v", changes2)
	}
}

func TestChangesSizeWithHardlinks(t *testing.T) {
	// TODO Windows. There may be a way of running this, but turning off for now
	// as createSampleDir uses symlinks.
	if runtime.GOOS == windows {
		t.Skip("hardlinks on Windows")
	}
	srcDir := t.TempDir()
	destDir := t.TempDir()

	creationSize, err := prepareUntarSourceDirectory(100, destDir, true)
	require.NoError(t, err)

	changes, err := ChangesDirs(destDir, &idtools.IDMappings{}, srcDir, &idtools.IDMappings{})
	require.NoError(t, err)

	got := ChangesSize(destDir, changes)
	if got != int64(creationSize) {
		t.Errorf("Expected %d bytes of changes, got %d", creationSize, got)
	}
}

func TestChangesSizeWithNoChanges(t *testing.T) {
	size := ChangesSize("/tmp", nil)
	if size != 0 {
		t.Fatalf("ChangesSizes with no changes should be 0, was %d", size)
	}
}

func TestChangesSizeWithOnlyDeleteChanges(t *testing.T) {
	changes := []Change{
		{Path: "deletedPath", Kind: ChangeDelete},
	}
	size := ChangesSize("/tmp", changes)
	if size != 0 {
		t.Fatalf("ChangesSizes with only delete changes should be 0, was %d", size)
	}
}

func TestChangesSize(t *testing.T) {
	parentPath := t.TempDir()
	addition := path.Join(parentPath, "addition")
	err := os.WriteFile(addition, []byte{0x01, 0x01, 0x01}, 0o744)
	require.NoError(t, err)
	modification := path.Join(parentPath, "modification")
	err = os.WriteFile(modification, []byte{0x01, 0x01, 0x01}, 0o744)
	require.NoError(t, err)

	changes := []Change{
		{Path: "addition", Kind: ChangeAdd},
		{Path: "modification", Kind: ChangeModify},
	}
	size := ChangesSize(parentPath, changes)
	if size != 6 {
		t.Fatalf("Expected 6 bytes of changes, got %d", size)
	}
}

func checkChanges(t *testing.T, expectedChanges, changes []Change) {
	sort.Sort(changesByPath(expectedChanges))
	sort.Sort(changesByPath(changes))
	for i := range max(len(changes), len(expectedChanges)) {
		if i >= len(expectedChanges) {
			t.Fatalf("unexpected change %s\n", changes[i].String())
		}
		if i >= len(changes) {
			t.Fatalf("no change for expected change %s\n", expectedChanges[i].String())
		}
		if changes[i].Path == expectedChanges[i].Path {
			if changes[i] != expectedChanges[i] {
				t.Fatalf("Wrong change for %s, expected %s, got %s\n", changes[i].Path, changes[i].String(), expectedChanges[i].String())
			}
		} else if changes[i].Path < expectedChanges[i].Path {
			t.Fatalf("unexpected change %s\n", changes[i].String())
		} else {
			t.Fatalf("no change for expected change %s != %s\n", expectedChanges[i].String(), changes[i].String())
		}
	}
}
