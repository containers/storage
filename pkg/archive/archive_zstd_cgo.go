// +build linux,cgo

package archive

import (
	"io"

	zstd "github.com/valyala/gozstd"
)

type wrapperZstdDecoder struct {
	decoder *zstd.Reader
}

func (w *wrapperZstdDecoder) Close() error {
	w.decoder.Release()
	return nil
}

func (w *wrapperZstdDecoder) DecodeAll(input, dst []byte) ([]byte, error) {
	return zstd.Decompress(dst, input)
}

func (w *wrapperZstdDecoder) Read(p []byte) (int, error) {
	return w.decoder.Read(p)
}

func (w *wrapperZstdDecoder) Reset(r io.Reader) error {
	w.decoder.Reset(r, nil)
	return nil
}

func (w *wrapperZstdDecoder) WriteTo(wr io.Writer) (int64, error) {
	return w.decoder.WriteTo(wr)
}

func zstdReader(buf io.Reader) (io.ReadCloser, error) {
	decoder := zstd.NewReader(buf)
	return &wrapperZstdDecoder{decoder: decoder}, nil
}

func zstdWriter(dest io.Writer) (io.WriteCloser, error) {
	return zstd.NewWriter(dest), nil
}
