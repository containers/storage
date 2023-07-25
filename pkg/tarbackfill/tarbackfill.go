package tarbackfill

import (
	"archive/tar"
	"io"
	"path"
	"strings"
)

// Reader wraps a tar.Reader so that if an item which would be read from it is
// in a directory which is not included in the archive, a specified Backfiller
// interface's Backfill() method will be called to supply a tar.Header which
// will be inserted into the stream just ahead of that item.
type Reader struct {
	*tar.Reader
	backfiller      Backfiller
	seen            map[string]struct{}
	queue           []*tar.Header
	currentIsQueued bool
	err             error
}

// Backfiller is a wrapper for Backfill, which can supply headers to insert
// into an archive which is on its way to being extracted.
type Backfiller interface {
	// Backfill either returns an entry for the passed-in path, nil if
	// no entry should be added to the stream, or an error if something
	// unexpected happened.
	Backfill(string) (*tar.Header, error)
}

// NewReaderWithBackfiller creates a new Reader reading from r, asking the
// passed-in Backfiller for information about parent directories which it
// hasn't seen yet.
func NewReaderWithBackfiller(r *tar.Reader, backfiller Backfiller) *Reader {
	reader := &Reader{
		Reader:     r,
		backfiller: backfiller,
		seen:       make(map[string]struct{}),
	}
	return reader
}

// Next returns either the next item from the archive we're filtering, or a
// synthesized entry for a directory that arguably should have been in that
// archive.
func (r *Reader) Next() (*tar.Header, error) {
	// Drain the queue first.
	if len(r.queue) > 0 {
		next, queue := r.queue[0], r.queue[1:]
		r.queue = queue
		r.currentIsQueued = len(r.queue) > 0
		return next, nil
	}
	// If we've hit the end of the archive, we've hit the end of the archive.
	r.currentIsQueued = false
	if r.err != nil {
		return nil, r.err
	}
	// Check what's next in the archive.
	hdr, err := r.Reader.Next()
	if err != nil {
		r.err = err
	}
	if hdr == nil {
		return hdr, err
	}
	for {
		// Trim off an initial or final path separator.
		name := strings.Trim(hdr.Name, "/")
		if hdr.Typeflag == tar.TypeDir {
			// Trim off an initial or final path separator, and
			// note that we won't need to supply it later.
			r.seen[name] = struct{}{}
		}
		// Figure out which directory this item is directly in.
		p := name
		dir, _ := path.Split(name)
		var newHdr *tar.Header
		for dir != p {
			var bfErr error
			dir = strings.Trim(dir, "/")
			// If we already saw that directory, no need to interfere (further).
			if _, ok := r.seen[dir]; dir == "" || ok || dir == name {
				return hdr, err
			}
			// Ask the backfiller what to do here.
			newHdr, bfErr = r.backfiller.Backfill(dir)
			if bfErr != nil {
				r.err = bfErr
				return nil, bfErr
			}
			if newHdr == nil {
				dir, _ = path.Split(dir)
				continue
			}
			// Make sure the Name looks right, then queue up the current entry.
			newHdr.Format = tar.FormatPAX
			newHdr.Name = strings.Trim(newHdr.Name, "/")
			if newHdr.Typeflag == tar.TypeDir {
				// We won't need to supply it later.
				r.seen[newHdr.Name] = struct{}{}
				newHdr.Name += "/"
			}
			r.queue = append([]*tar.Header{hdr}, r.queue...)
			hdr = newHdr
			r.currentIsQueued = true
			dir, _ = path.Split(dir)
		}
	}
}

// Read will either read from a real entry in the archive, or pretend very hard
// that an entry we inserted had no content.
func (r *Reader) Read(b []byte) (int, error) {
	if r.currentIsQueued {
		return 0, nil
	}
	return r.Reader.Read(b)
}

// NewIOReaderWithBackfiller creates a new ReadCloser for reading from a
// Reader, asking the passed-in Backfiller for parent directories of items in
// the archive that aren't in the archive.
func NewIOReaderWithBackfiller(reader io.Reader, backfiller Backfiller) io.ReadCloser {
	rc, wc := io.Pipe()
	go func() {
		r := tar.NewReader(reader)
		tr := NewReaderWithBackfiller(r, backfiller)
		tw := tar.NewWriter(wc)
		hdr, err := tr.Next()
		defer func() {
			closeErr := tw.Close()
			io.Copy(wc, reader)
			if err != nil {
				wc.CloseWithError(err)
			} else if closeErr != nil {
				wc.CloseWithError(closeErr)
			} else {
				wc.Close()
			}
		}()
		for hdr != nil {
			if writeError := tw.WriteHeader(hdr); writeError != nil {
				return
			}
			if err != nil {
				break
			}
			if hdr.Size != 0 {
				if _, err = io.Copy(tw, tr); err != nil {
					return
				}
			}
			hdr, err = tr.Next()
		}
	}()
	return rc
}
