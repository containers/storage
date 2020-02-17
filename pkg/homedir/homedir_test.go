package homedir

// Copyright 2013-2018 Docker, Inc.
// NOTE: this package has originally been copied from github.com/docker/docker.

import (
	"path/filepath"
	"testing"
)

func TestGet(t *testing.T) {
	home := Get()
	if home == "" {
		t.Fatal("returned home directory is empty")
	}

	if !filepath.IsAbs(home) {
		t.Fatalf("returned path is not absolute: %s", home)
	}
}

func TestGetShortcutString(t *testing.T) {
	shortcut := GetShortcutString()
	if shortcut == "" {
		t.Fatal("returned shortcut string is empty")
	}
}
