//go:build darwin || linux

package archive

import (
	"github.com/containers/storage/pkg/system"
)

// setExtendedAttribute sets an extended attribute on a file. On Linux and Darwin,
// extended attributes are supported via the xattr system calls.
func setExtendedAttribute(path string, xattrKey string, value []byte) error {
	return system.Lsetxattr(path, xattrKey, value, 0)
}
