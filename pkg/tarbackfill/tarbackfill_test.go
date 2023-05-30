package tarbackfill

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"testing"
	"time"

	"github.com/containers/storage/pkg/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTarByteSlice(headers []*tar.Header, trailerLength int) []byte {
	var buf bytes.Buffer
	block := make([]byte, 256)
	for i := 0; i < 256; i++ {
		block[i] = byte(i % 256)
	}
	tw := tar.NewWriter(&buf)
	for i := range headers {
		hdr := *headers[i]
		hdr.Format = tar.FormatPAX
		tw.WriteHeader(&hdr)
		if hdr.Size > 0 {
			written := int64(0)
			for written < hdr.Size {
				left := hdr.Size - written
				if left > int64(len(block)) {
					left = int64(len(block))
				}
				n, err := tw.Write(block[:int(left)])
				if err != nil {
					break
				}
				written += int64(n)
			}
		}
		tw.Flush()
	}
	tw.Close()
	padding := make([]byte, trailerLength) // some layer diffs have more trailing zeros than necessary, we need to preserve them
	buf.Write(padding)
	return buf.Bytes()
}

func consumeTar(t *testing.T, reader io.Reader, fn func(*tar.Header)) {
	t.Helper()
	t.Run("parse", func(t *testing.T) {
		tr := tar.NewReader(reader)
		hdr, err := tr.Next()
		for hdr != nil {
			if fn != nil {
				fn(hdr)
			}
			if hdr.Size != 0 {
				n, err := io.Copy(ioutil.Discard, tr)
				require.NoErrorf(t, err, "unexpected error copying entry payload for %q", hdr.Name)
				require.Equalf(t, hdr.Size, n, "payload for %q had unexpected length", hdr.Name)
			}
			if err != nil {
				break
			}
			hdr, err = tr.Next()
		}
		require.ErrorIs(t, err, io.EOF, "hit an error that wasn't EOF")
		_, err = io.Copy(io.Discard, reader)
		require.NoError(t, err, "while draining possible trailer")
	})
}

type backfillerLogger struct {
	t        *testing.T
	log      *[]string
	backfill bool
	mode     int64
	uid, gid int
	date     time.Time
}

func (b *backfillerLogger) Backfill(path string) (*tar.Header, error) {
	if !stringutils.InSlice(*(b.log), path) {
		*(b.log) = append(*(b.log), path)
		sort.Strings(*(b.log))
	}
	if b.backfill {
		return &tar.Header{Name: path, Typeflag: tar.TypeDir, Mode: b.mode, Uid: b.uid, Gid: b.gid, ModTime: b.date}, nil
	}
	return nil, nil
}

func newBackfillerLogger(t *testing.T, log *[]string, backfill bool, mode int64, uid, gid int, date time.Time) *backfillerLogger {
	return &backfillerLogger{t: t, log: log, backfill: backfill, mode: mode, uid: uid, gid: gid, date: date}
}

func TestNewIOReaderWithBackfiller(t *testing.T) {
	directoryMode := int64(0o750)
	directoryUid := 5
	directoryGid := 6
	now := time.Now().UTC()
	testCases := []struct {
		description string
		inputs      []*tar.Header
		backfills   []string
		outputs     []*tar.Header
	}{
		{
			description: "empty",
		},
		{
			description: "base",
			inputs: []*tar.Header{
				{
					Name:     "a",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      1,
					Gid:      1,
					Size:     2,
					ModTime:  now,
				},
			},
			outputs: []*tar.Header{
				{
					Name:     "a",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      1,
					Gid:      1,
					Size:     2,
					ModTime:  now,
				},
			},
		},
		{
			description: "topdir",
			inputs: []*tar.Header{
				{
					Name:     "a",
					Typeflag: tar.TypeDir,
					Mode:     0o750,
					Uid:      1,
					Gid:      1,
					Size:     0,
					ModTime:  now,
				},
			},
			outputs: []*tar.Header{
				{
					Name:     "a",
					Typeflag: tar.TypeDir,
					Mode:     0o750,
					Uid:      1,
					Gid:      1,
					Size:     0,
					ModTime:  now,
				},
			},
		},
		{
			description: "shallow",
			inputs: []*tar.Header{
				{
					Name:     "a/b",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      1,
					Gid:      2,
					Size:     1234,
					ModTime:  now,
				},
				{
					Name:     "a/c",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      3,
					Gid:      4,
					Size:     1234,
					ModTime:  now,
				},
				{
					Name:     "a/d",
					Typeflag: tar.TypeDir,
					Mode:     0o700,
					Uid:      5,
					Gid:      6,
					Size:     0,
					ModTime:  now,
				},
			},
			backfills: []string{
				"a",
			},
			outputs: []*tar.Header{
				{
					Name:     "a/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "a/b",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      1,
					Gid:      2,
					Size:     1234,
					ModTime:  now,
				},
				{
					Name:     "a/c",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      3,
					Gid:      4,
					Size:     1234,
					ModTime:  now,
				},
				{
					Name:     "a/d",
					Typeflag: tar.TypeDir,
					Mode:     0o700,
					Uid:      5,
					Gid:      6,
					Size:     0,
					ModTime:  now,
				},
			},
		},
		{
			description: "deep",
			inputs: []*tar.Header{
				{
					Name:     "a/c",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      3,
					Gid:      4,
					Size:     1234,
					ModTime:  now,
				},
				{
					Name:     "a/b/c/d/",
					Typeflag: tar.TypeDir,
					Mode:     0o700,
					Uid:      1,
					Gid:      2,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "a/b/c/d/e/f/g",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      3,
					Gid:      4,
					Size:     12346,
					ModTime:  now,
				},
				{
					Name:     "b/c/d/e/f/g/",
					Typeflag: tar.TypeDir,
					Mode:     0o711,
					Uid:      5,
					Gid:      6,
					Size:     0,
					ModTime:  now,
				},
			},
			backfills: []string{
				"a",
				"a/b",
				"a/b/c",
				"a/b/c/d/e",
				"a/b/c/d/e/f",
				"b",
				"b/c",
				"b/c/d",
				"b/c/d/e",
				"b/c/d/e/f",
			},
			outputs: []*tar.Header{
				{
					Name:     "a/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "a/c",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      1,
					Gid:      2,
					Size:     1234,
					ModTime:  now,
				},
				{
					Name:     "a/b/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "a/b/c/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "a/b/c/d/",
					Typeflag: tar.TypeDir,
					Mode:     0o700,
					Uid:      1,
					Gid:      2,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "a/b/c/d/e/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "a/b/c/d/e/f/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "a/b/c/d/e/f/g",
					Typeflag: tar.TypeReg,
					Mode:     0o644,
					Uid:      3,
					Gid:      4,
					Size:     12346,
					ModTime:  now,
				},
				{
					Name:     "b/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "b/c/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "b/c/d/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "b/c/d/e/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "b/c/d/e/f/",
					Typeflag: tar.TypeDir,
					Mode:     directoryMode,
					Uid:      directoryUid,
					Gid:      directoryGid,
					Size:     0,
					ModTime:  now,
				},
				{
					Name:     "b/c/d/e/f/g/",
					Typeflag: tar.TypeDir,
					Mode:     0o711,
					Uid:      5,
					Gid:      6,
					Size:     0,
					ModTime:  now,
				},
			},
		},
	}
	for testCase := range testCases {
		t.Run(testCases[testCase].description, func(t *testing.T) {
			for _, paddingSize := range []int{0, 512, 1024, 2048, 4096, 8192} {
				t.Run(fmt.Sprintf("paddingSize=%d", paddingSize), func(t *testing.T) {
					tarBytes := makeTarByteSlice(testCases[testCase].inputs, paddingSize)

					t.Run("basic", func(t *testing.T) {
						tarBytesReader := bytes.NewReader(tarBytes)
						consumeTar(t, tarBytesReader, nil)
						assert.Zero(t, tarBytesReader.Len())
					})

					t.Run("logged", func(t *testing.T) {
						var backfillLog []string
						tarBytesReader := bytes.NewReader(tarBytes)
						rc := NewIOReaderWithBackfiller(tarBytesReader, newBackfillerLogger(t, &backfillLog, false, 0o700, 1, 2, time.Time{}))
						defer rc.Close()
						consumeTar(t, rc, nil)
						require.Equal(t, testCases[testCase].backfills, backfillLog, "backfill not called exactly the right number of times")
						assert.Zero(t, tarBytesReader.Len())
					})

					t.Run("broken", func(t *testing.T) {
						var backfillLog []string
						tarBytesReader := bytes.NewReader(tarBytes)
						rc := NewIOReaderWithBackfiller(tarBytesReader, newBackfillerLogger(t, &backfillLog, false, directoryMode, directoryUid, directoryGid, now))
						defer rc.Close()
						consumeTar(t, rc, nil)
						assert.Zero(t, tarBytesReader.Len())
					})

					t.Run("filled", func(t *testing.T) {
						var backfillLog []string
						tarBytesReader := bytes.NewReader(tarBytes)
						rc := NewIOReaderWithBackfiller(tarBytesReader, newBackfillerLogger(t, &backfillLog, true, directoryMode, directoryUid, directoryGid, now))
						defer rc.Close()
						outputs := make([]*tar.Header, 0, len(testCases[testCase].inputs)+len(testCases[testCase].backfills))
						consumeTar(t, rc, func(hdr *tar.Header) { tmp := *hdr; hdr = &tmp; outputs = append(outputs, hdr) })
						require.Equal(t, len(testCases[testCase].outputs), len(outputs), "wrong number of output entries")
						assert.Zero(t, tarBytesReader.Len())
						if len(outputs) != 0 {
							for i := range outputs {
								expected := testCases[testCase].outputs[i]
								actual := outputs[i]
								require.EqualValuesf(t, expected.Name, actual.Name, "output %d name", i)
								require.EqualValuesf(t, expected.Mode, actual.Mode, "output %d mode", i)
								require.EqualValuesf(t, expected.Typeflag, actual.Typeflag, "output %d type", i)
								require.Truef(t, actual.ModTime.UTC().Equal(expected.ModTime.UTC()), "output %d (%q) date differs (%v != %v)", i, actual.Name, actual.ModTime.UTC(), expected.ModTime.UTC())
							}
						}
						require.Equal(t, testCases[testCase].backfills, backfillLog, "backfill not called exactly the right number of times")
					})
				})
			}
		})
	}
}
