package directory

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// Usage of an empty directory should be 0
func TestUsageEmpty(t *testing.T) {
	var dir string
	var err error
	if dir, err = ioutil.TempDir(os.TempDir(), "testUsageEmptyDirectory"); err != nil {
		t.Fatalf("failed to create directory: %s", err)
	}
	defer os.RemoveAll(dir)

	usage, _ := Usage(dir)
	expectSizeAndInodeCount(t, "empty directory", usage, &DiskUsage{
		Size:       0,
		InodeCount: 1,
	})
}

// Usage of one empty file should be 0
func TestUsageEmptyFile(t *testing.T) {
	var dir string
	var err error
	if dir, err = ioutil.TempDir(os.TempDir(), "testUsageEmptyFile"); err != nil {
		t.Fatalf("failed to create directory: %s", err)
	}
	defer os.RemoveAll(dir)

	var file *os.File
	if file, err = ioutil.TempFile(dir, "file"); err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	usage, _ := Usage(file.Name())
	expectSizeAndInodeCount(t, "one file", usage, &DiskUsage{
		Size:       0,
		InodeCount: 1,
	})
}

// Usage of a directory with one 5-byte file should be 5
func TestUsageNonemptyFile(t *testing.T) {
	var dir string
	var err error
	if dir, err = ioutil.TempDir(os.TempDir(), "testUsageNonemptyFile"); err != nil {
		t.Fatalf("failed to create directory: %s", err)
	}
	defer os.RemoveAll(dir)

	var file *os.File
	if file, err = ioutil.TempFile(dir, "file"); err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	d := []byte{97, 98, 99, 100, 101}
	file.Write(d)

	usage, _ := Usage(dir)
	expectSizeAndInodeCount(t, "directory with one 5-byte file", usage, &DiskUsage{
		Size:       5,
		InodeCount: 2,
	})
}

// Usage of an empty directory should be 0
func TestUsageEmptyDirectory(t *testing.T) {
	var dir string
	var err error
	if dir, err = ioutil.TempDir(os.TempDir(), "testUsageEmptyDirectory"); err != nil {
		t.Fatalf("failed to create directory: %s", err)
	}
	defer os.RemoveAll(dir)

	usage, _ := Usage(dir)
	expectSizeAndInodeCount(t, "one directory", usage, &DiskUsage{
		Size:       0,
		InodeCount: 1,
	})
}

// Usage of a directory with one empty directory should be 0
func TestUsageNestedDirectoryEmpty(t *testing.T) {
	var dir string
	var err error
	if dir, err = ioutil.TempDir(os.TempDir(), "testUsageNestedDirectoryEmpty"); err != nil {
		t.Fatalf("failed to create directory: %s", err)
	}
	defer os.RemoveAll(dir)
	if _, err = ioutil.TempDir(dir, "nested"); err != nil {
		t.Fatalf("failed to create nested directory: %s", err)
	}

	usage, _ := Usage(dir)
	expectSizeAndInodeCount(t, "directory with one empty directory", usage, &DiskUsage{
		Size:       0,
		InodeCount: 2,
	})
}

// Test directory with 1 file and 1 empty directory
func TestUsageFileAndNestedDirectoryEmpty(t *testing.T) {
	var dir string
	var err error
	if dir, err = ioutil.TempDir(os.TempDir(), "testUsageFileAndNestedDirectoryEmpty"); err != nil {
		t.Fatalf("failed to create directory: %s", err)
	}
	defer os.RemoveAll(dir)
	if _, err = ioutil.TempDir(dir, "nested"); err != nil {
		t.Fatalf("failed to create nested directory: %s", err)
	}

	var file *os.File
	if file, err = ioutil.TempFile(dir, "file"); err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	d := []byte{100, 111, 99, 107, 101, 114}
	file.Write(d)

	usage, _ := Usage(dir)
	expectSizeAndInodeCount(t, "directory with 6-byte file and empty directory", usage, &DiskUsage{
		Size:       6,
		InodeCount: 3,
	})
}

// Test directory with 1 file and 1 non-empty directory
func TestUsageFileAndNestedDirectoryNonempty(t *testing.T) {
	var dir, dirNested string
	var err error
	if dir, err = ioutil.TempDir(os.TempDir(), "TestUsageFileAndNestedDirectoryNonempty"); err != nil {
		t.Fatalf("failed to create directory: %s", err)
	}
	defer os.RemoveAll(dir)
	if dirNested, err = ioutil.TempDir(dir, "nested"); err != nil {
		t.Fatalf("failed to create nested directory: %s", err)
	}

	var file *os.File
	if file, err = ioutil.TempFile(dir, "file"); err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	data := []byte{100, 111, 99, 107, 101, 114}
	file.Write(data)

	var nestedFile *os.File
	if nestedFile, err = ioutil.TempFile(dirNested, "file"); err != nil {
		t.Fatalf("failed to create file in nested directory: %s", err)
	}

	nestedData := []byte{100, 111, 99, 107, 101, 114}
	nestedFile.Write(nestedData)

	usage, _ := Usage(dir)
	expectSizeAndInodeCount(t, "directory with 6-byte file and nested directory with 6-byte file", usage, &DiskUsage{
		Size:       12,
		InodeCount: 4,
	})
}

// Test migration of directory to a subdir underneath itself
func TestMoveToSubdir(t *testing.T) {
	var outerDir, subDir string
	var err error

	if outerDir, err = ioutil.TempDir(os.TempDir(), "TestMoveToSubdir"); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	defer os.RemoveAll(outerDir)

	if subDir, err = ioutil.TempDir(outerDir, "testSub"); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// write 4 temp files in the outer dir to get moved
	filesList := []string{"a", "b", "c", "d"}
	for _, fName := range filesList {
		if file, err := os.Create(filepath.Join(outerDir, fName)); err != nil {
			t.Fatalf("couldn't create temp file %q: %v", fName, err)
		} else {
			file.WriteString(fName)
			file.Close()
		}
	}

	if err = MoveToSubdir(outerDir, filepath.Base(subDir)); err != nil {
		t.Fatalf("Error during migration of content to subdirectory: %v", err)
	}
	// validate that the files were moved to the subdirectory
	infos, err := ioutil.ReadDir(subDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 4 {
		t.Fatalf("Should be four files in the subdir after the migration: actual length: %d", len(infos))
	}
	var results []string
	for _, info := range infos {
		results = append(results, info.Name())
	}
	sort.Sort(sort.StringSlice(results))
	if !reflect.DeepEqual(filesList, results) {
		t.Fatalf("Results after migration do not equal list of files: expected: %v, got: %v", filesList, results)
	}
}

// Test a non-existing directory
func TestUsageNonExistingDirectory(t *testing.T) {
	if _, err := Usage("/thisdirectoryshouldnotexist/TestUsageNonExistingDirectory"); err == nil {
		t.Fatalf("error is expected")
	}
}

// A helper function that tests expectation of inode count and dir size against
// the found usage.
func expectSizeAndInodeCount(t *testing.T, testName string, current, expected *DiskUsage) {
	if current.Size != expected.Size {
		t.Errorf("%s has size: %d, expected %d", testName, current.Size, expected.Size)
	}
	if current.InodeCount != expected.InodeCount {
		t.Errorf("%s has inode count: %d, expected %d", testName, current.InodeCount, expected.InodeCount)
	}
}
