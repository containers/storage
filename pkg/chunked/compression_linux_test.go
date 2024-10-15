package chunked

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbatts/tar-split/archive/tar"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"
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

	res, err := tarSizeFromTarSplit(tarSplit.Bytes())
	require.NoError(t, err)
	assert.Equal(t, expectedTarSize, res)
}
