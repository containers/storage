//go:build windows

package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileWithInvalidDest(t *testing.T) {
	// TODO Windows: This is currently failing. Not sure what has
	// recently changed in CopyWithTar as used to pass. Further investigation
	// is required.
	t.Skip("Currently fails")
	folder := t.TempDir()
	dest := "c:dest"
	srcFolder := filepath.Join(folder, "src")
	src := filepath.Join(folder, "src", "src")
	err := os.MkdirAll(srcFolder, 0o740)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(src, []byte("content"), 0o777)
	err = defaultCopyWithTar(src, dest)
	if err == nil {
		t.Fatalf("archiver.CopyWithTar should throw an error on invalid dest.")
	}
}

func TestCanonicalTarNameForPath(t *testing.T) {
	cases := []struct {
		in, expected string
		shouldFail   bool
	}{
		{"foo", "foo", false},
		{"foo/bar", "___", true}, // unix-styled windows path must fail
		{`foo\bar`, "foo/bar", false},
	}
	for _, v := range cases {
		if out, err := CanonicalTarNameForPath(v.in); err != nil && !v.shouldFail {
			t.Fatalf("cannot get canonical name for path: %s: %v", v.in, err)
		} else if v.shouldFail && err == nil {
			t.Fatalf("canonical path call should have failed with error. in=%s out=%s", v.in, out)
		} else if !v.shouldFail && out != v.expected {
			t.Fatalf("wrong canonical tar name. expected:%s got:%s", v.expected, out)
		}
	}
}

func TestCanonicalTarName(t *testing.T) {
	cases := []struct {
		in       string
		isDir    bool
		expected string
	}{
		{"foo", false, "foo"},
		{"foo", true, "foo/"},
		{`foo\bar`, false, "foo/bar"},
		{`foo\bar`, true, "foo/bar/"},
	}
	for _, v := range cases {
		if out, err := canonicalTarName(v.in, v.isDir); err != nil {
			t.Fatalf("cannot get canonical name for path: %s: %v", v.in, err)
		} else if out != v.expected {
			t.Fatalf("wrong canonical tar name. expected:%s got:%s", v.expected, out)
		}
	}
}

func TestChmodTarEntry(t *testing.T) {
	cases := []struct {
		in, expected os.FileMode
	}{
		{0o000, 0o111},
		{0o777, 0o755},
		{0o644, 0o755},
		{0o755, 0o755},
		{0o444, 0o555},
		{0o755 | os.ModeDir, 0o755 | os.ModeDir},
		{0o755 | os.ModeSymlink, 0o755 | os.ModeSymlink},
	}
	for _, v := range cases {
		if out := chmodTarEntry(v.in); out != v.expected {
			t.Fatalf("wrong chmod. expected:%v got:%v", v.expected, out)
		}
	}
}
