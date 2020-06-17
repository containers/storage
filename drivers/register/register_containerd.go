// +build !exclude_graphdriver_containerd,linux

package register

import (
	// register the containerd graphdriver
	_ "github.com/containers/storage/drivers/containerd"
)
