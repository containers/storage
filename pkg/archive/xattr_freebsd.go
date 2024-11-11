//go:build freebsd

package archive

import (
	"strings"
	"syscall"

	"github.com/containers/storage/pkg/system"
)

// setExtendedAttribute sets an extended attribute on a file. On FreeBSD, extended attributes are
// supported via the extattr system calls.
func setExtendedAttribute(path string, xattrKey string, value []byte) error {
	namespace, attrname, err := xattrToExtattr(xattrKey)
	if err != nil {
		return err
	}
	return system.ExtattrSetLink(path, namespace, attrname, value)
}

func xattrToExtattr(xattrname string) (namespace int, attrname string, err error) {
	namespaceMap := map[string]int{
		"user":   system.EXTATTR_NAMESPACE_USER,
		"system": system.EXTATTR_NAMESPACE_SYSTEM,
	}

	namespaceName, attrname, found := strings.Cut(xattrname, ".")
	if !found {
		return -1, "", syscall.ENOTSUP
	}

	namespace, ok := namespaceMap[namespaceName]
	if !ok {
		return -1, "", syscall.ENOTSUP
	}
	return namespace, attrname, nil
}
