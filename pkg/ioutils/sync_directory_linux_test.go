//go:build linux

package ioutils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncDirectoryContents(t *testing.T) {
	dir := t.TempDir()

	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	files := []string{
		filepath.Join(dir, "file1"),
		filepath.Join(nested, "file2"),
	}
	for _, file := range files {
		if err := os.WriteFile(file, []byte("storage-resilience"), 0o644); err != nil {
			t.Fatalf("write file %q: %v", file, err)
		}
	}

	if err := SyncDirectoryContents(dir); err != nil {
		t.Fatalf("SyncDirectoryContents: %v", err)
	}
}
