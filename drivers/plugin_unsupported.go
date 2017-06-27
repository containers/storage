// +build !experimental

package graphdriver

import (
	"github.com/pkg/errors"
)

func lookupPlugin(name, home string, opts []string) (Driver, error) {
	return nil, errors.Wrapf(ErrNotSupported, "plugin")
}
