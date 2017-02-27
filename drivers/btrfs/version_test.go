// +build linux,btrfs_version

package btrfs

import (
	"testing"
)

func TestLibVersion(t *testing.T) {
	if btrfsLibVersion() <= 0 {
		t.Errorf("expected output from btrfs lib version > 0")
	}
}
