//go:build linux || freebsd || solaris
// +build linux freebsd solaris

package graphtest

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"testing"

	graphdriver "github.com/containers/storage/drivers"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/stringid"
	"github.com/docker/go-units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

var (
	drv *Driver
)

const (
	defaultPerms  = os.FileMode(0555)
	modifiedPerms = os.FileMode(0711)
)

// Driver conforms to graphdriver.Driver interface and
// contains information such as root and reference count of the number of clients using it.
// This helps in testing drivers added into the framework.
type Driver struct {
	graphdriver.Driver
	root     string
	runroot  string
	refCount int
}

func newDriver(t testing.TB, name string, options []string) *Driver {
	root, err := os.MkdirTemp("", "storage-graphtest-")
	require.NoError(t, err)
	runroot, err := os.MkdirTemp("", "storage-graphtest-")
	require.NoError(t, err)

	require.NoError(t, os.MkdirAll(root, 0755))
	d, err := graphdriver.GetDriver(name, graphdriver.Options{DriverOptions: options, Root: root, RunRoot: runroot})
	if err != nil {
		t.Logf("graphdriver: %v\n", err)
		if errors.Is(err, graphdriver.ErrNotSupported) || errors.Is(err, graphdriver.ErrPrerequisites) || errors.Is(err, graphdriver.ErrIncompatibleFS) {
			t.Skipf("Driver %s not supported", name)
		}
		t.Fatal(err)
	}
	return &Driver{d, root, runroot, 1}
}

func cleanup(t testing.TB, d *Driver) {
	if err := drv.Cleanup(); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(d.runroot)
	os.RemoveAll(d.root)
}

// GetDriver create a new driver with given name or return an existing driver with the name updating the reference count.
func GetDriver(t testing.TB, name string, options ...string) graphdriver.Driver {
	if drv == nil {
		drv = newDriver(t, name, options)
	} else {
		drv.refCount++
	}
	return drv
}

// PutDriver removes the driver if it is no longer used and updates the reference count.
func PutDriver(t testing.TB) {
	if drv == nil {
		t.Skip("No driver to put!")
	}
	drv.refCount--
	if drv.refCount == 0 {
		cleanup(t, drv)
		drv = nil
	}
}

// DriverTestCreateEmpty creates a new image and verifies it is empty and the right metadata
func DriverTestCreateEmpty(t testing.TB, drivername string, driverOptions ...string) {
	driver := GetDriver(t, drivername, driverOptions...)
	defer PutDriver(t)

	err := driver.Create("empty", "", nil)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, driver.Remove("empty"))
	}()

	if !driver.Exists("empty") {
		t.Fatal("Newly created image doesn't exist")
	}

	dir, err := driver.Get("empty", graphdriver.MountOpts{})
	require.NoError(t, err)

	verifyFile(t, dir, defaultPerms|os.ModeDir, 0, 0)

	// Verify that the directory is empty
	fis, err := readDir(dir)
	require.NoError(t, err)
	assert.Len(t, fis, 0)

	driver.Put("empty")
}

// DriverTestCreateBase create a base driver and verify.
func DriverTestCreateBase(t testing.TB, drivername string, driverOptions ...string) {
	driver := GetDriver(t, drivername, driverOptions...)
	defer PutDriver(t)

	createBase(t, driver, "Base1")
	defer func() {
		require.NoError(t, driver.Remove("Base1"))
	}()
	verifyBase(t, driver, "Base1", defaultPerms)
}

// DriverTestCreateSnap Create a driver and snap and verify.
func DriverTestCreateSnap(t testing.TB, drivername string, driverOptions ...string) {
	driver := GetDriver(t, drivername, driverOptions...)
	defer PutDriver(t)

	createBase(t, driver, "Base2")
	defer func() {
		require.NoError(t, driver.Remove("Base2"))
	}()

	err := driver.Create("Snap2", "Base2", nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, driver.Remove("Snap2"))
	}()

	verifyBase(t, driver, "Snap2", defaultPerms)

	root, err := driver.Get("Snap2", graphdriver.MountOpts{})
	assert.NoError(t, err)
	err = os.Chmod(root, modifiedPerms)
	require.NoError(t, err)
	driver.Put("Snap2")

	err = driver.Create("SecondSnap", "Snap2", nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, driver.Remove("SecondSnap"))
	}()

	verifyBase(t, driver, "SecondSnap", modifiedPerms)
}

// DriverTestCreateFromTemplate Create a driver and template of a snap and verifies its
// contents.
func DriverTestCreateFromTemplate(t testing.TB, drivername string, driverOptions ...string) {
	driver := GetDriver(t, drivername, driverOptions...)
	defer PutDriver(t)

	createBase(t, driver, "Base3")
	defer func() {
		require.NoError(t, driver.Remove("Base3"))
	}()

	err := driver.Create("Snap3", "Base3", nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, driver.Remove("Snap3"))
	}()

	content := []byte("test content")
	if err := addFile(driver, "Snap3", "testfile.txt", content); err != nil {
		t.Fatal(err)
	}

	err = driver.CreateFromTemplate("FromTemplate", "Snap3", nil, "Base3", nil, nil, true)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, driver.Remove("FromTemplate"))
	}()

	err = driver.CreateFromTemplate("ROFromTemplate", "Snap3", nil, "Base3", nil, nil, false)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, driver.Remove("ROFromTemplate"))
	}()

	noChanges := []archive.Change{}

	changes, err := driver.Changes("FromTemplate", nil, "Snap3", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = checkChanges(noChanges, changes); err != nil {
		t.Fatal(err)
	}

	changes, err = driver.Changes("ROFromTemplate", nil, "Snap3", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = checkChanges(noChanges, changes); err != nil {
		t.Fatal(err)
	}

	if err := checkFile(driver, "FromTemplate", "testfile.txt", content); err != nil {
		t.Fatal(err)
	}
	if err := checkFile(driver, "ROFromTemplate", "testfile.txt", content); err != nil {
		t.Fatal(err)
	}
	if err := checkFile(driver, "Snap3", "testfile.txt", content); err != nil {
		t.Fatal(err)
	}

	expectedChanges := []archive.Change{{
		Path: "/testfile.txt",
		Kind: archive.ChangeAdd,
	}}

	changes, err = driver.Changes("Snap3", nil, "Base3", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = checkChanges(expectedChanges, changes); err != nil {
		t.Fatal(err)
	}

	changes, err = driver.Changes("FromTemplate", nil, "Base3", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = checkChanges(expectedChanges, changes); err != nil {
		t.Fatal(err)
	}

	changes, err = driver.Changes("ROFromTemplate", nil, "Base3", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if err = checkChanges(expectedChanges, changes); err != nil {
		t.Fatal(err)
	}

	verifyBase(t, driver, "Base3", defaultPerms)
}

// DriverTestDeepLayerRead reads a file from a lower layer under a given number of layers
func DriverTestDeepLayerRead(t testing.TB, layerCount int, drivername string, driverOptions ...string) {
	driver := GetDriver(t, drivername, driverOptions...)
	defer PutDriver(t)

	base := stringid.GenerateRandomID()
	if err := driver.Create(base, "", nil); err != nil {
		t.Fatal(err)
	}

	content := []byte("test content")
	if err := addFile(driver, base, "testfile.txt", content); err != nil {
		t.Fatal(err)
	}

	topLayer, err := addManyLayers(driver, base, layerCount)
	if err != nil {
		t.Fatal(err)
	}

	err = checkManyLayers(driver, topLayer, layerCount)
	if err != nil {
		t.Fatal(err)
	}

	if err := checkFile(driver, topLayer, "testfile.txt", content); err != nil {
		t.Fatal(err)
	}
}

// DriverTestDiffApply tests diffing and applying produces the same layer
func DriverTestDiffApply(t testing.TB, fileCount int, drivername string, driverOptions ...string) {
	driver := GetDriver(t, drivername, driverOptions...)
	defer PutDriver(t)
	base := stringid.GenerateRandomID()
	upper := stringid.GenerateRandomID()
	deleteFile := "file-remove.txt"
	deleteFileContent := []byte("This file should get removed in upper!")
	deleteDir := "var/lib"

	if err := driver.Create(base, "", nil); err != nil {
		t.Fatal(err)
	}

	if err := addManyFiles(driver, base, fileCount, 3); err != nil {
		t.Fatal(err)
	}

	if err := addFile(driver, base, deleteFile, deleteFileContent); err != nil {
		t.Fatal(err)
	}

	if err := addDirectory(driver, base, deleteDir); err != nil {
		t.Fatal(err)
	}

	if err := driver.Create(upper, base, nil); err != nil {
		t.Fatal(err)
	}

	if err := addManyFiles(driver, upper, fileCount, 6); err != nil {
		t.Fatal(err)
	}

	if err := removeAll(driver, upper, deleteFile, deleteDir); err != nil {
		t.Fatal(err)
	}

	diffSize, err := driver.DiffSize(upper, nil, "", nil, "")
	if err != nil {
		t.Fatal(err)
	}

	diff := stringid.GenerateRandomID()
	if err := driver.Create(diff, base, nil); err != nil {
		t.Fatal(err)
	}

	if err := checkManyFiles(driver, diff, fileCount, 3); err != nil {
		t.Fatal(err)
	}

	if err := checkFile(driver, diff, deleteFile, deleteFileContent); err != nil {
		t.Fatal(err)
	}

	arch, err := driver.Diff(upper, nil, base, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(nil)
	if _, err := buf.ReadFrom(arch); err != nil {
		t.Fatal(err)
	}
	if err := arch.Close(); err != nil {
		t.Fatal(err)
	}

	applyDiffSize, err := driver.ApplyDiff(diff, base, graphdriver.ApplyDiffOpts{Diff: bytes.NewReader(buf.Bytes())})
	if err != nil {
		t.Fatal(err)
	}

	if applyDiffSize != diffSize {
		t.Fatalf("Apply diff size different, got %d, expected %d", applyDiffSize, diffSize)
	}

	if err := checkManyFiles(driver, diff, fileCount, 6); err != nil {
		t.Fatal(err)
	}

	if err := checkFileRemoved(driver, diff, deleteFile); err != nil {
		t.Fatal(err)
	}

	if err := checkFileRemoved(driver, diff, deleteDir); err != nil {
		t.Fatal(err)
	}
}

// DriverTestChanges tests computed changes on a layer matches changes made
func DriverTestChanges(t testing.TB, drivername string, driverOptions ...string) {
	driver := GetDriver(t, drivername, driverOptions...)
	defer PutDriver(t)
	base := stringid.GenerateRandomID()
	upper := stringid.GenerateRandomID()
	if err := driver.Create(base, "", nil); err != nil {
		t.Fatal(err)
	}

	if err := addManyFiles(driver, base, 20, 3); err != nil {
		t.Fatal(err)
	}

	if err := driver.Create(upper, base, nil); err != nil {
		t.Fatal(err)
	}

	expectedChanges, err := changeManyFiles(driver, upper, 20, 6)
	if err != nil {
		t.Fatal(err)
	}

	changes, err := driver.Changes(upper, nil, base, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	if err = checkChanges(expectedChanges, changes); err != nil {
		t.Fatal(err)
	}
}

func writeRandomFile(path string, size uint64) error {
	data := make([]byte, size)

	rng := rand.New(rand.NewSource(0))
	if _, err := rng.Read(data); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0700)
}

// DriverTestSetQuota Create a driver and test setting quota.
func DriverTestSetQuota(t *testing.T, drivername string) {
	driver := GetDriver(t, drivername)
	defer PutDriver(t)

	createBase(t, driver, "Base4")
	createOpts := &graphdriver.CreateOpts{}
	createOpts.StorageOpt = make(map[string]string, 1)
	createOpts.StorageOpt["size"] = "50M"
	if err := driver.Create("quotaTest", "Base4", createOpts); err != nil {
		t.Fatal(err)
	}

	mountPath, err := driver.Get("quotaTest", graphdriver.MountOpts{})
	if err != nil {
		t.Fatal(err)
	}

	quota := uint64(50 * units.MiB)
	err = writeRandomFile(path.Join(mountPath, "file"), quota*2)
	if pathError, ok := err.(*os.PathError); ok && pathError.Err != unix.EDQUOT {
		t.Fatalf("expect write() to fail with %v, got %v", unix.EDQUOT, err)
	}

	if err := driver.Put("quotaTest"); err != nil {
		t.Fatal(err)
	}
}

// DriverTestEcho tests that we can diff a layer correctly, focusing on trouble spots that NaiveDiff doesn't have
func DriverTestEcho(t testing.TB, drivername string, driverOptions ...string) {
	driver := GetDriver(t, drivername, driverOptions...)
	defer PutDriver(t)
	var err error
	var root string
	components := 10

	for depth := 0; depth < components; depth++ {
		base := stringid.GenerateRandomID()
		second := stringid.GenerateRandomID()
		third := stringid.GenerateRandomID()

		if err := driver.Create(base, "", nil); err != nil {
			t.Fatal(err)
		}

		if root, err = driver.Get(base, graphdriver.MountOpts{}); err != nil {
			t.Fatal(err)
		}

		paths := []string{}
		path := "/"
		expectedChanges := []archive.Change{}
		for i := 0; i < components-1; i++ {
			path = filepath.Join(path, fmt.Sprintf("subdir%d", i+1))
			paths = append(paths, path)
			if err = os.Mkdir(filepath.Join(root, path), 0700); err != nil {
				t.Fatal(err)
			}
			expectedChanges = append(expectedChanges, archive.Change{Kind: archive.ChangeAdd, Path: path})
		}
		path = filepath.Join(path, "file")
		paths = append(paths, path)
		if err = os.WriteFile(filepath.Join(root, path), randomContent(128, int64(depth)), 0600); err != nil {
			t.Fatal(err)
		}
		expectedChanges = append(expectedChanges, archive.Change{Kind: archive.ChangeAdd, Path: path})

		changes, err := driver.Changes(base, nil, "", nil, "")
		if err != nil {
			t.Fatal(err)
		}

		if err = checkChanges(expectedChanges, changes); err != nil {
			t.Fatal(err)
		}

		if err := driver.Create(second, base, nil); err != nil {
			t.Fatal(err)
		}

		if root, err = driver.Get(second, graphdriver.MountOpts{}); err != nil {
			t.Fatal(err)
		}

		if err = os.RemoveAll(filepath.Join(root, paths[depth])); err != nil {
			t.Fatal(err)
		}
		expectedChanges = []archive.Change{}
		for i := 0; i < depth; i++ {
			expectedChanges = append(expectedChanges, archive.Change{Kind: archive.ChangeModify, Path: paths[i]})
		}
		expectedChanges = append(expectedChanges, archive.Change{Kind: archive.ChangeDelete, Path: paths[depth]})

		changes, err = driver.Changes(second, nil, base, nil, "")
		if err != nil {
			t.Fatal(err)
		}

		if err = checkChanges(expectedChanges, changes); err != nil {
			t.Fatal(err)
		}

		if err = driver.Create(third, second, nil); err != nil {
			t.Fatal(err)
		}

		if root, err = driver.Get(third, graphdriver.MountOpts{}); err != nil {
			t.Fatal(err)
		}

		expectedChanges = []archive.Change{}
		for i := 0; i < depth; i++ {
			expectedChanges = append(expectedChanges, archive.Change{Kind: archive.ChangeModify, Path: paths[i]})
		}
		for i := depth; i < components-1; i++ {
			if err = os.Mkdir(filepath.Join(root, paths[i]), 0700); err != nil {
				t.Fatal(err)
			}
			expectedChanges = append(expectedChanges, archive.Change{Kind: archive.ChangeAdd, Path: paths[i]})
		}
		if err = os.WriteFile(filepath.Join(root, paths[len(paths)-1]), randomContent(128, int64(depth)), 0600); err != nil {
			t.Fatal(err)
		}
		expectedChanges = append(expectedChanges, archive.Change{Kind: archive.ChangeAdd, Path: paths[len(paths)-1]})

		changes, err = driver.Changes(third, nil, second, nil, "")
		if err != nil {
			t.Fatal(err)
		}

		if err = checkChanges(expectedChanges, changes); err != nil {
			t.Fatal(err)
		}

		err = driver.Put(third)
		if err != nil {
			t.Fatal(err)
		}
		err = driver.Put(second)
		if err != nil {
			t.Fatal(err)
		}
		err = driver.Put(base)
		if err != nil {
			t.Fatal(err)
		}
	}
}
