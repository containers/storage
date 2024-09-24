//go:build freebsd

package system

import (
	"os"
	"path/filepath"
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

func TestFileFlags(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "append")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Lchflags(file, UF_READONLY); err != nil {
		t.Fatal(err)
	}
	s, err := Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if s.Flags() != UF_READONLY {
		t.Fatal("got invalid flags")
	}
}
