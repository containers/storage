package compressor

// NOTE: This is used from github.com/containers/image by callers that
// don't otherwise use containers/storage, so don't make this depend on any
// larger software like the graph drivers.

import (
	"bufio"
	"encoding/base64"
	"io"
	"io/ioutil"

	"github.com/containers/storage/pkg/chunked/internal"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/opencontainers/go-digest"
	"github.com/vbatts/tar-split/archive/tar"
)

const RollsumBits = 16

type rollingChecksumReader struct {
	reader  *bufio.Reader
	closed  bool
	rollsum *RollSum

	WrittenOut int64
}

func (rc *rollingChecksumReader) Read(b []byte) (bool, int, error) {
	if rc.closed {
		return false, 0, io.EOF
	}
	for i := 0; i < len(b); i++ {
		n, err := rc.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				rc.closed = true
				if i == 0 {
					return false, 0, err
				}
				return false, i, nil
			}
			// Report any other error type
			return false, -1, err
		}
		b[i] = n
		rc.WrittenOut++
		rc.rollsum.Roll(n)
		if rc.rollsum.OnSplitWithBits(RollsumBits) {
			return true, i + 1, nil
		}
	}
	return false, len(b), nil
}

type chunk struct {
	ChunkOffset int64
	Offset      int64
	Checksum    string
	ChunkSize   int64
}

func writeZstdChunkedStream(destFile io.Writer, outMetadata map[string]string, reader io.Reader, level int) error {
	// total written so far.  Used to retrieve partial offsets in the file
	dest := ioutils.NewWriteCounter(destFile)

	tr := tar.NewReader(reader)
	tr.RawAccounting = true

	buf := make([]byte, 4096)

	zstdWriter, err := internal.ZstdWriterWithLevel(dest, level)
	if err != nil {
		return err
	}
	defer func() {
		if zstdWriter != nil {
			zstdWriter.Close()
			zstdWriter.Release()
		}
	}()

	restartCompression := func() (int64, error) {
		var offset int64
		if zstdWriter != nil {
			if err := zstdWriter.Close(); err != nil {
				return 0, err
			}
			offset = dest.Count
			zstdWriter.Reset(dest, nil, level)
		}
		return offset, nil
	}

	var metadata []internal.FileMetadata
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		rawBytes := tr.RawBytes()
		if _, err := zstdWriter.Write(rawBytes); err != nil {
			return err
		}

		payloadDigester := digest.Canonical.Digester()
		chunkDigester := digest.Canonical.Digester()

		// Now handle the payload, if any
		startOffset := int64(0)
		lastOffset := int64(0)
		lastChunkOffset := int64(0)

		checksum := ""

		chunks := []chunk{}

		rcReader := &rollingChecksumReader{
			reader:  bufio.NewReader(tr),
			rollsum: NewRollSum(),
		}

		payloadDest := io.MultiWriter(payloadDigester.Hash(), chunkDigester.Hash(), zstdWriter)
		for {
			mustSplit, read, errRead := rcReader.Read(buf)
			if errRead != nil && errRead != io.EOF {
				return err
			}
			// restart the compression only if there is a payload.
			if read > 0 {
				if startOffset == 0 {
					startOffset, err = restartCompression()
					if err != nil {
						return err
					}
					lastOffset = startOffset
				}

				if _, err := payloadDest.Write(buf[:read]); err != nil {
					return err
				}
			}
			if (mustSplit || errRead == io.EOF) && startOffset > 0 {
				off, err := restartCompression()
				if err != nil {
					return err
				}

				chunks = append(chunks, chunk{
					ChunkOffset: lastChunkOffset,
					Offset:      lastOffset,
					Checksum:    chunkDigester.Digest().String(),
					ChunkSize:   rcReader.WrittenOut - lastChunkOffset,
				})

				lastOffset = off
				lastChunkOffset = rcReader.WrittenOut
				chunkDigester = digest.Canonical.Digester()
				payloadDest = io.MultiWriter(payloadDigester.Hash(), chunkDigester.Hash(), zstdWriter)
			}
			if errRead == io.EOF {
				if startOffset > 0 {
					checksum = payloadDigester.Digest().String()
				}
				break
			}
		}

		typ, err := internal.GetType(hdr.Typeflag)
		if err != nil {
			return err
		}
		xattrs := make(map[string]string)
		for k, v := range hdr.Xattrs {
			xattrs[k] = base64.StdEncoding.EncodeToString([]byte(v))
		}
		entries := []internal.FileMetadata{
			{
				Type:       typ,
				Name:       hdr.Name,
				Linkname:   hdr.Linkname,
				Mode:       hdr.Mode,
				Size:       hdr.Size,
				UID:        hdr.Uid,
				GID:        hdr.Gid,
				ModTime:    &hdr.ModTime,
				AccessTime: &hdr.AccessTime,
				ChangeTime: &hdr.ChangeTime,
				Devmajor:   hdr.Devmajor,
				Devminor:   hdr.Devminor,
				Xattrs:     xattrs,
				Digest:     checksum,
				Offset:     startOffset,
				EndOffset:  lastOffset,
			},
		}
		for i := 1; i < len(chunks); i++ {
			entries = append(entries, internal.FileMetadata{
				Type:        internal.TypeChunk,
				Name:        hdr.Name,
				ChunkOffset: chunks[i].ChunkOffset,
			})
		}
		if len(chunks) > 1 {
			for i := range chunks {
				entries[i].ChunkSize = chunks[i].ChunkSize
				entries[i].Offset = chunks[i].Offset
				entries[i].ChunkDigest = chunks[i].Checksum
			}
		}
		metadata = append(metadata, entries...)
	}

	rawBytes := tr.RawBytes()
	if _, err := zstdWriter.Write(rawBytes); err != nil {
		return err
	}
	if err := zstdWriter.Flush(); err != nil {
		return err
	}
	if err := zstdWriter.Close(); err != nil {
		return err
	}
	zstdWriter = nil

	return internal.WriteZstdChunkedManifest(dest, outMetadata, uint64(dest.Count), metadata, level)
}

type zstdChunkedWriter struct {
	tarSplitOut *io.PipeWriter
	tarSplitErr chan error
}

func (w zstdChunkedWriter) Close() error {
	err := <-w.tarSplitErr
	if err != nil {
		w.tarSplitOut.Close()
		return err
	}
	return w.tarSplitOut.Close()
}

func (w zstdChunkedWriter) Write(p []byte) (int, error) {
	select {
	case err := <-w.tarSplitErr:
		w.tarSplitOut.Close()
		return 0, err
	default:
		return w.tarSplitOut.Write(p)
	}
}

// zstdChunkedWriterWithLevel writes a zstd compressed tarball where each file is
// compressed separately so it can be addressed separately.  Idea based on CRFS:
// https://github.com/google/crfs
// The difference with CRFS is that the zstd compression is used instead of gzip.
// The reason for it is that zstd supports embedding metadata ignored by the decoder
// as part of the compressed stream.
// A manifest json file with all the metadata is appended at the end of the tarball
// stream, using zstd skippable frames.
// The final file will look like:
// [FILE_1][FILE_2]..[FILE_N][SKIPPABLE FRAME 1][SKIPPABLE FRAME 2]
// Where:
// [FILE_N]: [ZSTD HEADER][TAR HEADER][PAYLOAD FILE_N][ZSTD FOOTER]
// [SKIPPABLE FRAME 1]: [ZSTD SKIPPABLE FRAME, SIZE=MANIFEST LENGTH][MANIFEST]
// [SKIPPABLE FRAME 2]: [ZSTD SKIPPABLE FRAME, SIZE=16][MANIFEST_OFFSET][MANIFEST_LENGTH][MANIFEST_LENGTH_UNCOMPRESSED][MANIFEST_TYPE][CHUNKED_ZSTD_MAGIC_NUMBER]
// MANIFEST_OFFSET, MANIFEST_LENGTH, MANIFEST_LENGTH_UNCOMPRESSED and CHUNKED_ZSTD_MAGIC_NUMBER are 64 bits unsigned in little endian format.
func zstdChunkedWriterWithLevel(out io.Writer, metadata map[string]string, level int) (io.WriteCloser, error) {
	ch := make(chan error, 1)
	r, w := io.Pipe()

	go func() {
		ch <- writeZstdChunkedStream(out, metadata, r, level)
		io.Copy(ioutil.Discard, r)
		r.Close()
		close(ch)
	}()

	return zstdChunkedWriter{
		tarSplitOut: w,
		tarSplitErr: ch,
	}, nil
}

// ZstdCompressor is a CompressorFunc for the zstd compression algorithm.
func ZstdCompressor(r io.Writer, metadata map[string]string, level *int) (io.WriteCloser, error) {
	if level == nil {
		l := 10
		level = &l
	}

	return zstdChunkedWriterWithLevel(r, metadata, *level)
}
