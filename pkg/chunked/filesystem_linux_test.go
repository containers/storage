package chunked

import (
	"bytes"
	"io"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/chunked/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopCloser struct {
	*bytes.Reader
}

func (nopCloser) Close() error {
	return nil
}

func TestSeekableFileGetBlobAt(t *testing.T) {
	content := []byte("Hello, World!")

	br := bytes.NewReader(content)
	reader := nopCloser{br}

	sf := newSeekableFile(reader)
	chunks := []ImageSourceChunk{
		{Offset: 0, Length: 5},
		{Offset: 7, Length: 5},
	}

	streams, errs, err := sf.GetBlobAt(chunks)
	assert.NoError(t, err)

	expectedContents := [][]byte{
		[]byte("Hello"),
		[]byte("World"),
	}

	i := 0
	for stream := range streams {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(stream)
		require.NoError(t, err)

		require.Equal(t, expectedContents[i], buf.Bytes())
		i++
	}

	err, ok := <-errs
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestDoHardLink(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := createTempFile(t, tmpDir, "source")
	defer srcFile.Close()
	srcFd := int(srcFile.Fd())

	destDir := t.TempDir()
	destDirFd, err := syscall.Open(destDir, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	require.NoError(t, err)
	defer syscall.Close(destDirFd)

	destBase := "dest-file"
	err = doHardLink(destDirFd, srcFd, destBase)
	require.NoError(t, err)

	// an existing file is unlinked first
	err = doHardLink(destDirFd, srcFd, destBase)
	assert.NoError(t, err)

	err = doHardLink(destDirFd, -1, destBase)
	assert.Error(t, err)

	err = doHardLink(-1, srcFd, destBase)
	assert.Error(t, err)
}

func TestAppendHole(t *testing.T) {
	tmpDir := t.TempDir()

	tmpFile := createTempFile(t, tmpDir, "file-with-holes")
	defer tmpFile.Close()

	fd := int(tmpFile.Fd())

	size := int64(1024)

	err := appendHole(fd, tmpFile.Name(), size)
	assert.NoError(t, err, "Appending hole failed")

	fileSize, err := syscall.Seek(fd, 0, io.SeekEnd)
	assert.NoError(t, err)
	assert.Equal(t, size, fileSize, "File size is not as expected")
}

func TestSafeMkdir(t *testing.T) {
	rootDir := t.TempDir()
	dirName := "../dir"

	rootFile, err := os.Open(rootDir)
	require.NoError(t, err)
	defer rootFile.Close()

	rootFd := int(rootFile.Fd())

	metadata := fileMetadata{
		FileMetadata: internal.FileMetadata{
			Type: internal.TypeDir,
			Mode: 0o755,
		},
	}
	options := &archive.TarOptions{
		// Allow the test to run without privileges
		IgnoreChownErrors: true,
	}

	err = safeMkdir(rootFd, 0o755, dirName, &metadata, options)
	require.NoError(t, err)

	dir, err := openFileUnderRoot(rootFd, dirName, syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	require.NoError(t, err)
	err = dir.Close()
	assert.NoError(t, err)
}

func TestSafeLink(t *testing.T) {
	linkName := "a-hard-link"

	rootDir := t.TempDir()

	rootFile, err := os.Open(rootDir)
	require.NoError(t, err)
	defer rootFile.Close()

	rootFd := int(rootFile.Fd())

	file := createTempFile(t, rootDir, "an-existing-file")
	existingFile := path.Base(file.Name())
	err = file.Close()
	assert.NoError(t, err)

	metadata := fileMetadata{
		FileMetadata: internal.FileMetadata{
			Name: linkName,
			// try to create outside the root
			Linkname: "../../" + existingFile,
			Type:     internal.TypeReg,
			Mode:     0o755,
		},
	}
	options := &archive.TarOptions{
		// Allow the test to run without privileges
		IgnoreChownErrors: true,
	}

	err = safeLink(rootFd, 0o755, &metadata, options)
	require.NoError(t, err)

	// validate it was created
	newFile, err := openFileUnderRoot(rootFd, linkName, syscall.O_RDONLY, 0)
	require.NoError(t, err)

	st := syscall.Stat_t{}
	err = syscall.Fstat(int(newFile.Fd()), &st)
	assert.NoError(t, err)

	// We need this conversion on ARM64
	assert.Equal(t, uint64(st.Nlink), uint64(2))

	err = newFile.Close()
	assert.NoError(t, err)
}

func TestSafeSymlink(t *testing.T) {
	linkName := "a-hard-link"

	rootDir := t.TempDir()

	rootFile, err := os.Open(rootDir)
	require.NoError(t, err)
	defer rootFile.Close()

	rootFd := int(rootFile.Fd())

	file := createTempFile(t, rootDir, "an-existing-file")
	st := syscall.Stat_t{}
	err = syscall.Fstat(int(file.Fd()), &st)
	assert.NoError(t, err)

	err = file.Close()
	assert.NoError(t, err)

	existingFile := path.Base(file.Name())

	metadata := fileMetadata{
		FileMetadata: internal.FileMetadata{
			Name: linkName,
			// try to create outside the root
			Linkname: "../../" + existingFile,
			Type:     internal.TypeReg,
			Mode:     0o755,
		},
	}

	err = safeSymlink(rootFd, &metadata)
	require.NoError(t, err)

	// validate it was created
	newFile, err := openFileUnderRoot(rootFd, linkName, syscall.O_RDONLY, 0)
	require.NoError(t, err)

	st2 := syscall.Stat_t{}
	err = syscall.Fstat(int(newFile.Fd()), &st2)
	require.NoError(t, err)

	// validate that the opened file is the same as the original file that was
	// created earlier.  Compare the inode and device numbers.
	assert.Equal(t, st.Dev, st2.Dev)
	assert.Equal(t, st.Ino, st2.Ino)

	err = newFile.Close()
	assert.NoError(t, err)
}

func TestOpenOrCreateDirUnderRoot(t *testing.T) {
	rootDir := t.TempDir()
	dirName := "dir"

	rootFile, err := os.Open(rootDir)
	require.NoError(t, err)
	defer rootFile.Close()

	rootFd := int(rootFile.Fd())

	// try to create a directory outside the root
	dir, err := openOrCreateDirUnderRoot(rootFd, "../../"+dirName, 0o755)
	require.NoError(t, err)
	err = dir.Close()
	assert.NoError(t, err)

	dir, err = openFileUnderRoot(rootFd, dirName, syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	require.NoError(t, err)
	err = dir.Close()
	require.NoError(t, err)
}

func TestCopyFileContent(t *testing.T) {
	rootDir := t.TempDir()

	rootFile, err := os.Open(rootDir)
	require.NoError(t, err)
	defer rootFile.Close()

	rootFd := int(rootFile.Fd())

	file := createTempFile(t, rootDir, "an-existing-file")
	defer file.Close()

	size, err := file.Write([]byte("Hello, World!"))
	require.NoError(t, err)

	_, err = file.Seek(0, io.SeekStart)
	assert.NoError(t, err)

	st := syscall.Stat_t{}
	err = syscall.Fstat(int(file.Fd()), &st)
	require.NoError(t, err)

	metadata := fileMetadata{
		FileMetadata: internal.FileMetadata{
			Name: "new-file",
			Type: internal.TypeDir,
			Mode: 0o755,
		},
	}

	newFile, newSize, err := copyFileContent(int(file.Fd()), &metadata, rootFd, 0o755, false)
	require.NoError(t, err)

	assert.Equal(t, size, int(newSize))

	st2 := syscall.Stat_t{}
	err = syscall.Fstat(int(newFile.Fd()), &st2)
	require.NoError(t, err)

	err = newFile.Close()
	require.NoError(t, err)

	// the file was copied without hard links, the inodes must be different
	assert.Equal(t, st.Dev, st2.Dev)
	assert.NotEqual(t, st.Ino, st2.Ino)

	metadataCopyHardLinks := fileMetadata{
		FileMetadata: internal.FileMetadata{
			Name: "new-file2",
			Type: internal.TypeDir,
			Mode: 0o755,
		},
	}

	newFile, newSize, err = copyFileContent(int(file.Fd()), &metadataCopyHardLinks, rootFd, 0o755, true)
	require.NoError(t, err)
	assert.Nil(t, newFile)

	// validate it was created as an inode
	newFile, err = openFileUnderRoot(rootFd, metadataCopyHardLinks.FileMetadata.Name, syscall.O_RDONLY, 0)
	require.NoError(t, err)

	assert.Equal(t, size, int(newSize))

	st2 = syscall.Stat_t{}
	err = syscall.Fstat(int(newFile.Fd()), &st2)
	require.NoError(t, err)

	err = newFile.Close()
	require.NoError(t, err)

	// the file was copied with hard links, the inodes must be equal
	assert.Equal(t, st.Dev, st2.Dev)
	assert.Equal(t, st.Ino, st2.Ino)
}

func createTempFile(t *testing.T, dir, name string) *os.File {
	tmpFile, err := os.CreateTemp(dir, name)
	require.NoError(t, err)
	return tmpFile
}
