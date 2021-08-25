// +build linux

package chunked

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/containers/storage/pkg/chunked/internal"
)

func TestIsZstdChunkedFrameMagic(t *testing.T) {
	b := append(internal.ZstdChunkedFrameMagic[:], make([]byte, 200)...)
	if !isZstdChunkedFrameMagic(b) {
		t.Fatal("Chunked frame magic not found")
	}
	// change a byte
	b[0] = -b[0]
	if isZstdChunkedFrameMagic(b) {
		t.Fatal("Invalid chunked frame magic found")
	}
}

type seekable struct {
	data   []byte
	offset uint64
	length uint64
	t      *testing.T
}

func (s seekable) GetBlobAt(req []ImageSourceChunk) (chan io.ReadCloser, chan error, error) {
	if len(req) != 1 {
		s.t.Fatal("Requested more than one chunk")
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
		m <- ioutil.NopCloser(bytes.NewReader(s.data))
		close(m)
		close(e)
	}()

	return m, e, nil
}

var someFiles = []internal.FileMetadata{
	{
		Type: "dir",
		Name: "/foo",
		Mode: 0755,
		Size: 0,
	},
	{
		Type:        "reg",
		Name:        "/foo/bar",
		Mode:        0755,
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
		Mode:        0755,
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

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	if err := internal.WriteZstdChunkedManifest(writer, annotations, offsetManifest, someFiles[:], 9); err != nil {
		t.Error(err)
	}
	if err := writer.Flush(); err != nil {
		t.Error(err)
	}

	offsetMetadata := annotations[internal.ManifestInfoKey]
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
	if manifestType != internal.ManifestTypeCRFS {
		t.Fatalf("Invalid manifest type %d", manifestType)
	}
	if b.Len() == 0 {
		t.Fatal("no manifest written")
	}

	data := b.Bytes()[offset-offsetManifest:]
	s := seekable{
		data:   data,
		offset: offset,
		length: length,
		t:      t,
	}

	manifest, _, err := readZstdChunkedManifest(s, 8192, annotations)
	if err != nil {
		t.Error(err)
	}

	var toc internal.TOC
	if err := json.Unmarshal(manifest, &toc); err != nil {
		t.Error(err)
	}

	if toc.Version != 1 {
		t.Fatal("Invalid manifest version generated")
	}
	if len(toc.Entries) != len(someFiles) {
		t.Fatal("Manifest mismatch")
	}
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
	for k, v := range internal.TarTypes {
		r, err := internal.GetType(k)
		if err != nil {
			t.Error(err)
		}
		if r != v {
			t.Fatal("Invalid GetType conversion")
		}
	}
	if _, err := internal.GetType(byte('Z')); err == nil {
		t.Fatal("Invalid GetType conversion")
	}
}
