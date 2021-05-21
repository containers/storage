// +build linux freebsd

package system

import (
	"os"
	"syscall"
	"testing"
)

// TestFromStatT tests fromStatT for a tempfile
func TestFromStatT(t *testing.T) {
	file, _, _, dir := prepareFiles(t)
	defer os.RemoveAll(dir)

	stat := &syscall.Stat_t{}
	if err := syscall.Lstat(file, stat); err != nil {
		t.Fatal(err)
	}

	s, err := fromStatT(stat)
	if err != nil {
		t.Fatal(err)
	}

	if stat.Mode != s.Mode() {
		t.Fatal("got invalid mode")
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
	if stat.Mtim != s.Mtim() {
		t.Fatal("got invalid mtim")
	}
	if stat.Dev != s.Dev() {
		t.Fatal("got invalid dev")
	}
	if stat.Nlink != s.Nlink() {
		t.Fatal("got invalid nlink")
	}
	if stat.Ino != s.Ino() {
		t.Fatal("got invalid inode")
	}
}
