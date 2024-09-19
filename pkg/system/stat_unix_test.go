//go:build linux || freebsd

package system

import (
	"syscall"
	"testing"
)

// TestFromStatT tests fromStatT for a tempfile
func TestFromStatT(t *testing.T) {
	file, _, _ := prepareFiles(t)

	stat := &syscall.Stat_t{}
	if err := syscall.Lstat(file, stat); err != nil {
		t.Fatal(err)
	}

	s, err := fromStatT(stat)
	if err != nil {
		t.Fatal(err)
	}

	if stat.Uid != s.UID() {
		t.Fatal("got invalid uid")
	}
	if stat.Gid != s.GID() {
		t.Fatal("got invalid gid")
	}
	if stat.Rdev != s.Rdev() {
		t.Fatal("got invalid rdev")
	}
	// Types for Mode can vary and not all platforms have an Mtim
	// member in Stat_t
	platformTestFromStatT(t, stat, s)
}
