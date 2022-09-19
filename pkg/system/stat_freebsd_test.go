//go:build freebsd
// +build freebsd

package system

import (
	"syscall"
	"testing"
)

// TestFromStatT tests fromStatT for a tempfile
func platformTestFromStatT(t *testing.T, stat *syscall.Stat_t, s *StatT) {
	if stat.Mode != uint16(s.Mode()) {
		t.Fatal("got invalid mode")
	}
	if stat.Mtimespec != s.Mtim() {
		t.Fatal("got invalid mtim")
	}
}
