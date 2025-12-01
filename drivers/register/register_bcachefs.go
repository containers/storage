//go:build !exclude_graphdriver_bcachefs && linux

package register

import (
	_ "github.com/containers/storage/drivers/bcachefs"
)
