package chunked

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbatts/tar-split/archive/tar"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"
	"golang.org/x/sys/unix"
)

func TestTarSizeFromTarSplit(t *testing.T) {
	var tarball bytes.Buffer
	tarWriter := tar.NewWriter(&tarball)
	for _, e := range someFiles {
		tf, err := typeToTarType(e.Type)
		require.NoError(t, err)
		err = tarWriter.WriteHeader(&tar.Header{
			Typeflag: tf,
			Name:     e.Name,
			Size:     e.Size,
			Mode:     e.Mode,
		})
		require.NoError(t, err)
		data := make([]byte, e.Size)
		_, err = tarWriter.Write(data)
		require.NoError(t, err)
	}
	err := tarWriter.Close()
	require.NoError(t, err)
	expectedTarSize := int64(tarball.Len())

	var tarSplit bytes.Buffer
	tsReader, err := asm.NewInputTarStream(&tarball, storage.NewJSONPacker(&tarSplit), storage.NewDiscardFilePutter())
	require.NoError(t, err)
	_, err = io.Copy(io.Discard, tsReader)
	require.NoError(t, err)

	res, err := tarSizeFromTarSplit(&tarSplit)
	require.NoError(t, err)
	assert.Equal(t, expectedTarSize, res)
}

func TestOpenTmpFile(t *testing.T) {
	for i := 0; i < 1000; i++ {
		// scope for cleanup
		f := func(fn func(tmpDir string) (int, error)) {
			fd, err := fn(t.TempDir())
			assert.NoError(t, err)
			defer unix.Close(fd)

			path, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", fd))
			assert.NoError(t, err)

			// the path under /proc/self/fd/$FD has the prefix "(deleted)" when the file
			// is unlinked
			assert.Contains(t, path, "(deleted)")
		}
		f(openTmpFile)
		f(openTmpFileNoTmpFile)
	}
}
