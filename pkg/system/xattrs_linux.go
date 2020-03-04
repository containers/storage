package system

import (
	"bytes"

	"golang.org/x/sys/unix"
)

const (
	// Value is larger than the maximum size allowed
	E2BIG unix.Errno = unix.E2BIG

	// Operation not supported
	EOPNOTSUPP unix.Errno = unix.EOPNOTSUPP
)

// Lgetxattr retrieves the value of the extended attribute identified by attr
// and associated with the given path in the file system.
// Returns a []byte slice if the xattr is set and nil otherwise.
func Lgetxattr(path string, attr string) ([]byte, error) {
	// Start with a 128 length byte array
	dest := make([]byte, 128)
	sz, errno := unix.Lgetxattr(path, attr, dest)

	for errno == unix.ERANGE {
		// Buffer too small, use zero-sized buffer to get the actual size
		sz, errno = unix.Lgetxattr(path, attr, []byte{})
		if errno != nil {
			return nil, errno
		}
		dest = make([]byte, sz)
		sz, errno = unix.Lgetxattr(path, attr, dest)
	}

	switch {
	case errno == unix.ENODATA:
		return nil, nil
	case errno != nil:
		return nil, errno
	}

	return dest[:sz], nil
}

// Lsetxattr sets the value of the extended attribute identified by attr
// and associated with the given path in the file system.
func Lsetxattr(path string, attr string, data []byte, flags int) error {
	return unix.Lsetxattr(path, attr, data, flags)
}

// Llistxattr lists extended attributes associated with the given path
// in the file system.
func Llistxattr(path string) ([]string, error) {
	var dest []byte

	for {
		sz, err := unix.Llistxattr(path, dest)
		if err != nil {
			return nil, err
		}

		if sz > len(dest) {
			dest = make([]byte, sz)
		} else {
			dest = dest[:sz]
			break
		}
	}

	var attrs []string
	for _, token := range bytes.Split(dest, []byte{0}) {
		if len(token) > 0 {
			attrs = append(attrs, string(token))
		}
	}

	return attrs, nil
}
