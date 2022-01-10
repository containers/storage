// +build !linux !cgo

package archive

import (
	"fmt"
	"io"
)

type wrapperZstdDecoder struct {
}

func (w *wrapperZstdDecoder) Close() error {
	return fmt.Errorf("zstd not supported without cgo")
}

func (w *wrapperZstdDecoder) DecodeAll(input, dst []byte) ([]byte, error) {
	return nil, fmt.Errorf("zstd not supported without cgo")
}

func (w *wrapperZstdDecoder) Read(p []byte) (int, error) {
	return -1, fmt.Errorf("zstd not supported without cgo")
}

func (w *wrapperZstdDecoder) Reset(r io.Reader) error {
	return fmt.Errorf("zstd not supported without cgo")
}

func (w *wrapperZstdDecoder) WriteTo(wr io.Writer) (int64, error) {
	return -1, fmt.Errorf("zstd not supported without cgo")
}

func zstdReader(buf io.Reader) (io.ReadCloser, error) {
	return nil, fmt.Errorf("zstd not supported without cgo")
}

func zstdWriter(dest io.Writer) (io.WriteCloser, error) {
	return nil, fmt.Errorf("zstd not supported without cgo")
}
