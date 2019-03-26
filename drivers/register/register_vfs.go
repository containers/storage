// +build !exclude_graphdriver_vfs

package register

import (
	// register vfs
	_ "github.com/containers/storage/drivers/vfs"
)
