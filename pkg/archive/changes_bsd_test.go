//go:build freebsd

package archive

import (
	"os"
	"path"
	"testing"

	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/system"
	"github.com/stretchr/testify/require"
)

// Verify that file flag changes are reported
func TestChangeFileFlags(t *testing.T) {
	src := t.TempDir()
	createSampleDir(t, src)

	dst := src + "-copy"
	err := copyDir(src, dst)
	require.NoError(t, err)
	file1 := path.Join(dst, "dir1/file1-1")
	err = system.Lchflags(file1, system.UF_READONLY)
	require.NoError(t, err)

	changes, err := ChangesDirs(dst, &idtools.IDMappings{}, src, &idtools.IDMappings{})
	require.NoError(t, err)

	expectedChanges := []Change{
		{"/dir1", ChangeModify},
		{"/dir1/file1-1", ChangeModify},
	}
	checkChanges(t, expectedChanges, changes)
}

// Verify that file flag changes are copied
func TestCopyFileFlags(t *testing.T) {
	src := t.TempDir()
	createSampleDir(t, src)
	file1 := path.Join(src, "dir1/file1-1")
	err := system.Lchflags(file1, system.UF_READONLY)
	require.NoError(t, err)

	dst := src + "-copy"
	err = copyDir(src, dst)
	require.NoError(t, err)

	changes, err := ChangesDirs(dst, &idtools.IDMappings{}, src, &idtools.IDMappings{})
	require.NoError(t, err)

	if len(changes) != 0 {
		t.Fatalf("Changes with no difference should have detect no changes, but detected %d", len(changes))
	}
}

// Make sure we can apply changes to an immutable file, including deleting
func TestApplyToImmutable(t *testing.T) {
	// Make a directory with an immutable file
	src := t.TempDir()
	createSampleDir(t, src)
	file1 := path.Join(src, "dir1/file1-1")
	file2 := path.Join(src, "dir1/file1-2")
	require.NoError(t, os.Chmod(file1, 0o777))
	require.NoError(t, system.Lchflags(file1, system.SF_IMMUTABLE))
	require.NoError(t, system.Lchflags(file2, system.SF_IMMUTABLE))

	// Copy it, and change file1, delete file2
	dst := src + "-copy"
	err := copyDir(src, dst)
	require.NoError(t, err)
	file1 = path.Join(dst, "dir1/file1-1")
	file2 = path.Join(dst, "dir1/file1-2")
	require.NoError(t, system.Lchflags(file1, 0))
	require.NoError(t, os.Chmod(file1, 0o666))
	require.NoError(t, system.Lchflags(file2, 0))
	require.NoError(t, os.RemoveAll(file2))

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
