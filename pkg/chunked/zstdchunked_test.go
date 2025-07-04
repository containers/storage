//go:build linux

package chunked

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/containers/storage/pkg/chunked/internal/minimal"
	"github.com/containers/storage/pkg/chunked/toc"
	"github.com/klauspost/compress/zstd"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbatts/tar-split/archive/tar"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"
)

type seekable struct {
	data         []byte
	tarSplitData []byte
	offset       uint64
	length       uint64
	t            *testing.T
}

func (s seekable) GetBlobAt(req []ImageSourceChunk) (chan io.ReadCloser, chan error, error) {
	if len(req) != 2 {
		s.t.Fatal("Requested more than two chunks")
	}
	if req[0].Offset != s.offset {
		s.t.Fatal("Invalid offset requested")
	}
	if req[0].Length != s.length {
		s.t.Fatal("Invalid length requested")
	}

	m := make(chan io.ReadCloser)
	e := make(chan error)

	go func() {
		m <- io.NopCloser(bytes.NewReader(s.data))
		m <- io.NopCloser(bytes.NewReader(s.tarSplitData))
		close(m)
		close(e)
	}()

	return m, e, nil
}

var someFiles = []minimal.FileMetadata{
	{
		Type: "dir",
		Name: "/foo",
		Mode: 0o755,
		Size: 0,
	},
	{
		Type:        "reg",
		Name:        "/foo/bar",
		Mode:        0o755,
		Size:        10,
		Digest:      "sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
		Offset:      100,
		EndOffset:   110,
		ChunkSize:   10,
		ChunkDigest: "sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
		ChunkOffset: 0,
	},
	{
		Type:        "reg",
		Name:        "/foo/baz",
		Mode:        0o755,
		Size:        12,
		Digest:      "sha256:6f0378f21a495f5c13247317d158e9d51da45a5bf68fc2f366e450deafdc8302",
		Offset:      200,
		EndOffset:   212,
		ChunkSize:   12,
		ChunkDigest: "sha256:6f0378f21a495f5c13247317d158e9d51da45a5bf68fc2f366e450deafdc8302",
		ChunkOffset: 0,
	},
}

func TestGenerateAndParseManifest(t *testing.T) {
	annotations := make(map[string]string)
	offsetManifest := uint64(100000)

	var tsTarball bytes.Buffer
	tsTarW := tar.NewWriter(&tsTarball)
	for _, e := range someFiles {
		tf, err := typeToTarType(e.Type)
		require.NoError(t, err)
		err = tsTarW.WriteHeader(&tar.Header{
			Typeflag: tf,
			Name:     e.Name,
			Size:     e.Size,
			Mode:     e.Mode,
		})
		require.NoError(t, err)
		data := make([]byte, e.Size)
		_, err = tsTarW.Write(data)
		require.NoError(t, err)
	}
	err := tsTarW.Close()
	require.NoError(t, err)
	var tarSplitUncompressed bytes.Buffer
	tsReader, err := asm.NewInputTarStream(&tsTarball, storage.NewJSONPacker(&tarSplitUncompressed), storage.NewDiscardFilePutter())
	require.NoError(t, err)
	_, err = io.Copy(io.Discard, tsReader)
	require.NoError(t, err)

	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Error(err)
	}
	defer encoder.Close()
	tarSplitCompressedData := encoder.EncodeAll(tarSplitUncompressed.Bytes(), nil)

	ts := minimal.TarSplitData{
		Data:             tarSplitCompressedData,
		Digest:           digest.Canonical.FromBytes(tarSplitCompressedData),
		UncompressedSize: 9,
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	createZstdWriter := func(dest io.Writer) (minimal.ZstdWriter, error) {
		return minimal.ZstdWriterWithLevel(dest, 9)
	}

	if err := minimal.WriteZstdChunkedManifest(writer, annotations, offsetManifest, &ts, someFiles[:], createZstdWriter); err != nil {
		t.Error(err)
	}
	if err := writer.Flush(); err != nil {
		t.Error(err)
	}

	offsetMetadata := annotations[minimal.ManifestInfoKey]
	if offsetMetadata == "" {
		t.Fatal("Annotation not found")
	}

	var offset, length, lengthUncompressed, manifestType uint64
	if _, err := fmt.Sscanf(offsetMetadata, "%d:%d:%d:%d", &offset, &length, &lengthUncompressed, &manifestType); err != nil {
		t.Error(err)
	}

	if offset != offsetManifest+8 {
		t.Fatalf("Invalid offset %d", offset)
	}
	if manifestType != minimal.ManifestTypeCRFS {
		t.Fatalf("Invalid manifest type %d", manifestType)
	}
	if b.Len() == 0 {
		t.Fatal("no manifest written")
	}

	var tarSplitOffset, tarSplitLength, tarSplitUncompressedLength uint64
	tarSplitMetadata := annotations[minimal.TarSplitInfoKey]
	if _, err := fmt.Sscanf(tarSplitMetadata, "%d:%d:%d", &tarSplitOffset, &tarSplitLength, &tarSplitUncompressedLength); err != nil {
		t.Error(err)
	}

	if tarSplitOffset != offsetManifest+length+16 {
		t.Fatalf("Invalid tar split offset %d, expected %d", tarSplitOffset, offsetManifest+length+16)
	}

	data := b.Bytes()[offset-offsetManifest : offset-offsetManifest+length][:]
	tarSplitData := b.Bytes()[tarSplitOffset-offsetManifest : tarSplitOffset-offsetManifest+tarSplitLength][:]
	s := seekable{
		data:         data,
		tarSplitData: tarSplitData,
		offset:       offset,
		length:       length,
		t:            t,
	}

	tocDigest, err := toc.GetTOCDigest(annotations)
	require.NoError(t, err)
	require.NotNil(t, tocDigest)
	manifest, decodedTOC, _, _, err := readZstdChunkedManifest(t.TempDir(), s, *tocDigest, annotations, true)
	require.NoError(t, err)

	var toc minimal.TOC
	if err := json.Unmarshal(manifest, &toc); err != nil {
		t.Error(err)
	}

	if toc.Version != 1 {
		t.Fatal("Invalid manifest version generated")
	}
	if len(toc.Entries) != len(someFiles) {
		t.Fatal("Manifest mismatch")
	}
	assert.Equal(t, toc, *decodedTOC)
}

func TestGetTarType(t *testing.T) {
	for k, v := range typesToTar {
		r, err := typeToTarType(k)
		if err != nil {
			t.Error(err)
		}
		if r != v {
			t.Fatal("Invalid typeToTarType conversion")
		}
	}
	if _, err := typeToTarType("FOO"); err == nil {
		t.Fatal("Invalid typeToTarType conversion")
	}
	for k, v := range minimal.TarTypes {
		r, err := minimal.GetType(k)
		if err != nil {
			t.Error(err)
		}
		if r != v {
			t.Fatal("Invalid GetType conversion")
		}
	}
	if _, err := minimal.GetType(byte('Z')); err == nil {
		t.Fatal("Invalid GetType conversion")
	}
}
