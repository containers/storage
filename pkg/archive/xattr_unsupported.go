//go:build !darwin && !linux && !freebsd

package archive

import (
	"github.com/containers/storage/pkg/system"
)

// setExtendedAttribute is not supported on this platform.
func setExtendedAttribute(path string, xattrKey string, value []byte) error {
	return system.ErrNotSupportedPlatform
}
