package chunked

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	graphdriver "github.com/containers/storage/drivers"
	"github.com/stretchr/testify/assert"
)

const jsonTOC = `
{
  "version": 1,
  "entries": [
    {
      "type": "symlink",
      "name": "bin",
      "linkName": "usr/bin",
      "mode": 511,
      "modtime": "1970-01-01T01:00:00+01:00",
      "accesstime": "0001-01-01T00:00:00Z",
      "changetime": "0001-01-01T00:00:00Z"
    },
    {
      "type": "dir",
      "name": "usr/bin",
      "mode": 511,
      "modtime": "2022-01-07T12:36:43+01:00",
      "accesstime": "0001-01-01T00:00:00Z",
      "changetime": "0001-01-01T00:00:00Z"
    },
    {
      "type": "reg",
      "name": "usr/bin/foo",
      "mode": 511,
      "size": 103867,
      "modtime": "1970-01-01T01:00:00+01:00",
      "accesstime": "0001-01-01T00:00:00Z",
      "changetime": "0001-01-01T00:00:00Z",
      "digest": "sha256:99fe908c699dc068438b23e28319cadff1f2153c3043bafb8e83a430bba0a2c6",
      "offset": 94149,
      "endOffset": 120135,
      "chunkSize": 17615,
      "chunkDigest": "sha256:2ce0d0f8eb2aa93d13007097763e4459c814c8d0e859e5a57465af924169b544"
    },
    {
      "type": "chunk",
      "name": "usr/bin/foo",
      "offset": 99939,
      "chunkSize": 86252,
      "chunkOffset": 17615,
      "chunkDigest": "sha256:2a9d3f1b6b37abc8bb35eb8fa98b893a2a2447bcb01184c3bafc8c6b40da099d"
    },
    {
      "type": "reg",
      "name": "usr/lib/systemd/system/system-systemd\\x2dcryptsetup.slice",
      "mode": 420,
      "size": 468,
      "modtime": "2024-03-03T18:04:57+01:00",
      "accesstime": "0001-01-01T00:00:00Z",
      "changetime": "0001-01-01T00:00:00Z",
      "digest": "sha256:68dc6e85631e077f2bc751352459823844911b93b7ba2afd95d96c893222bb50",
      "offset": 148185424,
      "endOffset": 148185753
    },
    {
      "type": "reg",
      "name": "usr/lib/systemd/system/system-systemd\\x2dcryptsetup-hardlink.slice",
      "linkName": "usr/lib/systemd/system/system-systemd\\x2dcryptsetup.slice"
    }
  ]
}
`

func TestPrepareMetadata(t *testing.T) {
	toc, err := prepareCacheFile([]byte(jsonTOC), graphdriver.DifferOutputFormatDir)
	if err != nil {
		t.Errorf("got error from prepareCacheFile: %v", err)
	}
	if len(toc) != 4 {
		t.Error("prepareCacheFile returns the wrong length")
	}
}

func TestPrepareMetadataFlat(t *testing.T) {
	toc, err := prepareCacheFile([]byte(jsonTOC), graphdriver.DifferOutputFormatFlat)
	if err != nil {
		t.Errorf("got error from prepareCacheFile: %v", err)
	}
	for _, e := range toc {
		if len(strings.Split(e.Name, "/")) != 2 {
			t.Error("prepareCacheFile returns the wrong number of path elements for flat directories")
		}
		if len(filepath.Dir(e.Name)) != 2 {
			t.Error("prepareCacheFile returns the wrong path for flat directories")
		}
	}
}

type bigDataToBuffer struct {
	buf    *bytes.Buffer
	id     string
	key    string
	called bool
}

func (b *bigDataToBuffer) SetLayerBigData(id, key string, data io.Reader) error {
	b.id = id
	b.key = key
	if b.called {
		return fmt.Errorf("SetLayerBigData already called once")
	}
	b.called = true
	_, err := io.Copy(b.buf, data)
	return err
}

func findTag(digest string, cacheFile *cacheFile) (string, uint64, uint64) {
	binaryDigest, err := makeBinaryDigest(digest)
	if err != nil {
		return "", 0, 0
	}
	if len(binaryDigest) != cacheFile.digestLen {
		return "", 0, 0
	}
	found, off, len := findBinaryTag(binaryDigest, cacheFile)
	if found {
		return digest, off, len
	}
	return "", 0, 0
}

func TestWriteCache(t *testing.T) {
	toc, err := prepareCacheFile([]byte(jsonTOC), graphdriver.DifferOutputFormatDir)
	if err != nil {
		t.Errorf("got error from prepareCacheFile: %v", err)
	}

	dest := bigDataToBuffer{
		buf: bytes.NewBuffer(nil),
	}
	cache, err := writeCache([]byte(jsonTOC), graphdriver.DifferOutputFormatDir, "foobar", &dest)
	if err != nil {
		t.Errorf("got error from writeCache: %v", err)
	}
	if digest, _, _ := findTag("sha256:99fe908c699dc068438b23e28319cadff1f2153c3043bafb8e83a430bba0a2c2", cache); digest != "" {
		t.Error("a present tag was not found")
	}

	for _, r := range toc {
		if r.Digest != "" {
			// find the element in the cache by the digest checksum
			digest, off, lenTag := findTag(r.Digest, cache)
			if digest == "" {
				t.Error("file tag not found")
			}
			if digest != r.Digest {
				t.Error("wrong file found")
			}
			location := cache.vdata[off : off+lenTag]
			parts := strings.SplitN(string(location), ":", 3)

			assert.Equal(t, len(parts), 3)
			offFile, err := strconv.ParseInt(parts[0], 10, 64)
			assert.NoError(t, err)
			fileSize, err := strconv.ParseInt(parts[1], 10, 64)
			assert.NoError(t, err)

			assert.Equal(t, fileSize, int64(r.Size))
			assert.Equal(t, offFile, int64(0))

			fingerprint, err := calculateHardLinkFingerprint(r)
			if err != nil {
				t.Errorf("got error from writeCache: %v", err)
			}

			// find the element in the cache by the hardlink fingerprint
			digest, off, lenTag = findTag(fingerprint, cache)
			if digest == "" {
				t.Error("file tag not found")
			}
			if digest != fingerprint {
				t.Error("wrong file found")
			}
			location = cache.vdata[off : off+lenTag]
			parts = strings.SplitN(string(location), ":", 3)

			assert.Equal(t, len(parts), 3)
			offFile, err = strconv.ParseInt(parts[0], 10, 64)
			assert.NoError(t, err)
			fileSize, err = strconv.ParseInt(parts[1], 10, 64)
			assert.NoError(t, err)

			assert.Equal(t, fileSize, int64(r.Size))
			assert.Equal(t, offFile, int64(0))
		}
		if r.ChunkDigest != "" {
			// find the element in the cache by the chunk digest checksum
			digest, off, len := findTag(r.ChunkDigest, cache)
			if digest == "" {
				t.Error("chunk tag not found")
			}
			if digest != r.ChunkDigest {
				t.Error("wrong digest found")
			}
			expectedLocation := generateFileLocation(0, uint64(r.ChunkOffset), uint64(r.ChunkSize))
			location := cache.vdata[off : off+len]
			if !bytes.Equal(location, expectedLocation) {
				t.Errorf("wrong file found %q instead of %q", location, expectedLocation)
			}
		}
	}
}

func TestReadCache(t *testing.T) {
	dest := bigDataToBuffer{
		buf: bytes.NewBuffer(nil),
	}
	cache, err := writeCache([]byte(jsonTOC), graphdriver.DifferOutputFormatDir, "foobar", &dest)
	if err != nil {
		t.Errorf("got error from writeCache: %v", err)
	}

	cacheRead, err := readCacheFileFromMemory(dest.buf.Bytes())
	if err != nil {
		t.Errorf("got error from readMetadataFromCache: %v", err)
	}
	if !reflect.DeepEqual(cache, cacheRead) {
		t.Errorf("read a different struct than what was written")
	}
}

func TestUnmarshalToc(t *testing.T) {
	toc, err := unmarshalToc([]byte(jsonTOC))
	assert.NoError(t, err)
	assert.Equal(t, 6, len(toc.Entries))

	_, err = unmarshalToc([]byte(jsonTOC + "        \n\n\n\n    "))
	assert.NoError(t, err)
	_, err = unmarshalToc([]byte(jsonTOC + "aaaa"))
	assert.Error(t, err)
	_, err = unmarshalToc([]byte(jsonTOC + ","))
	assert.Error(t, err)
	_, err = unmarshalToc([]byte(jsonTOC + "{}"))
	assert.Error(t, err)
	_, err = unmarshalToc([]byte(jsonTOC + "[]"))
	assert.Error(t, err)
	_, err = unmarshalToc([]byte(jsonTOC + "\"aaaa\""))
	assert.Error(t, err)
	_, err = unmarshalToc([]byte(jsonTOC + "123"))
	assert.Error(t, err)
	assert.Equal(t, toc.Entries[4].Name, "usr/lib/systemd/system/system-systemd\\x2dcryptsetup.slice", "invalid name escaped")
	assert.Equal(t, toc.Entries[5].Name, "usr/lib/systemd/system/system-systemd\\x2dcryptsetup-hardlink.slice", "invalid name escaped")
	assert.Equal(t, toc.Entries[5].Linkname, "usr/lib/systemd/system/system-systemd\\x2dcryptsetup.slice", "invalid link name escaped")
}

func TestMakeBinaryDigest(t *testing.T) {
	binDigest, err := makeBinaryDigest("sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03")
	assert.NoError(t, err)
	expected := []byte{0x73, 0x68, 0x61, 0x32, 0x35, 0x36, 0x3a, 0x58, 0x91, 0xb5, 0xb5, 0x22, 0xd5, 0xdf, 0x8, 0x6d, 0xf, 0xf0, 0xb1, 0x10, 0xfb, 0xd9, 0xd2, 0x1b, 0xb4, 0xfc, 0x71, 0x63, 0xaf, 0x34, 0xd0, 0x82, 0x86, 0xa2, 0xe8, 0x46, 0xf6, 0xbe, 0x3}
	assert.Equal(t, expected, binDigest)

	_, err = makeBinaryDigest("sha256:foo")
	assert.Error(t, err)

	_, err = makeBinaryDigest("noAlgorithm")
	assert.Error(t, err)
}
