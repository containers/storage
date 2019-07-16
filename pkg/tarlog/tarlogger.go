package tarlog

import (
	"archive/tar"
	"io"
)

type tarLogger struct {
	w *io.PipeWriter
}

// NewLogger returns a writer that, when a tar archive is written to it, calls
// `logger` for each file header it encounters in the archive.
func NewLogger(logger func(*tar.Header)) io.WriteCloser {
	reader, writer := io.Pipe()
	go func() {
		r := tar.NewReader(reader)
		hdr, err := r.Next()
		for err == nil {
			logger(hdr)
			hdr, err = r.Next()
		}
		reader.Close()
	}()
	return &tarLogger{w: writer}
}

func (t *tarLogger) Write(b []byte) (int, error) {
	return t.w.Write(b)
}

func (t *tarLogger) Close() error {
	return t.w.Close()
}
