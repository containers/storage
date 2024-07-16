package chunked

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/containers/storage/pkg/chunked/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbatts/tar-split/archive/tar"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"
)

func createTestTarheader(index int, typeFlag byte, size int64) tar.Header {
	n := (index + 1) * 100 // Use predictable, but distinct, values for all headers

	res := tar.Header{
		Typeflag:   typeFlag,
		Name:       fmt.Sprintf("name%d", n),
		Size:       size,
		Mode:       int64(n + 1),
		Uid:        n + 2,
		Gid:        n + 3,
		Uname:      fmt.Sprintf("user%d", n),
		Gname:      fmt.Sprintf("group%d", n),
		ModTime:    time.Unix(int64(n+4), 0),
		AccessTime: time.Unix(int64(n+5), 0),
		ChangeTime: time.Unix(int64(n+6), 0),
		PAXRecords: map[string]string{fmt.Sprintf("key%d", n): fmt.Sprintf("value%d", n)},
		Format:     tar.FormatPAX, // We must set a format, in the default one AccessTime and ChangeTime are discarded.
	}
	switch res.Typeflag {
	case tar.TypeLink, tar.TypeSymlink:
		res.Linkname = fmt.Sprintf("link%d", n)
	case tar.TypeChar, tar.TypeBlock:
		res.Devmajor = int64(n + 7)
		res.Devminor = int64(n + 8)
	}
	return res
}

func TestIterateTarSplit(t *testing.T) {
	entries := []struct {
		typeFlag byte
		size     int64
	}{
		{tar.TypeReg, 0},
		{tar.TypeReg, 1},
		{tar.TypeReg, 511},
		{tar.TypeReg, 512},
		{tar.TypeReg, 513},
		{tar.TypeLink, 0},
		{tar.TypeSymlink, 0},
		{tar.TypeChar, 0},
		{tar.TypeBlock, 0},
		{tar.TypeDir, 0},
		{tar.TypeFifo, 0},
	}

	var tarball bytes.Buffer
	var expected []tar.Header
	w := tar.NewWriter(&tarball)
	for i, e := range entries {
		hdr := createTestTarheader(i, e.typeFlag, e.size)
		err := w.WriteHeader(&hdr)
		require.NoError(t, err)
		data := make([]byte, e.size)
		_, err = w.Write(data)
		require.NoError(t, err)
		expected = append(expected, hdr)
	}
	err := w.Close()
	require.NoError(t, err)

	var tarSplit bytes.Buffer
	tsReader, err := asm.NewInputTarStream(&tarball, storage.NewJSONPacker(&tarSplit), storage.NewDiscardFilePutter())
	require.NoError(t, err)
	_, err = io.Copy(io.Discard, tsReader)
	require.NoError(t, err)

	var actual []tar.Header
	err = iterateTarSplit(tarSplit.Bytes(), func(hdr *tar.Header) error {
		actual = append(actual, *hdr)
		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, len(expected), len(actual))
	for i := range expected {
		// We would have to open-code an equality comparison of time.Time values; instead, convert to FileMetadata,
		// because we already have that implemented for that type — and because it provides a tiny bit of code coverage
		// testing for ensureFileMetadataAttributesMatch.
		expected1, err := internal.NewFileMetadata(&expected[i])
		require.NoError(t, err, i)
		actual1, err := internal.NewFileMetadata(&actual[i])
		require.NoError(t, err, i)
		err = ensureFileMetadataAttributesMatch(&expected1, &actual1)
		assert.NoError(t, err, i)
	}
}
