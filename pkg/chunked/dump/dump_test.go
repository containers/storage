package dump

import (
	"bytes"
	"testing"
	"time"

	"github.com/containers/storage/pkg/chunked/internal"
)

func TestEscaped(t *testing.T) {
	tests := []struct {
		input  string
		escape int
		want   string
	}{
		{"12345", 0, "12345"},
		{"", 0, ""},
		{"\n", 0, "\\n"},
		{"\r", 0, "\\r"},
		{"\t", 0, "\\t"},
		{"\\", 0, "\\\\"},
		{"=", 0, "="},
		{"foo=bar", ESCAPE_EQUAL, "foo\\x3dbar"},
		{"-", ESCAPE_LONE_DASH, "\\x2d"},
		{"\n", NOESCAPE_SPACE, "\\n"},
		{" ", 0, "\\x20"},
		{" ", NOESCAPE_SPACE, " "},
		{"\t", NOESCAPE_SPACE, "\\t"},
		{"\n\t", NOESCAPE_SPACE, "\\n\\t"},
		{"Hello World!", 0, "Hello\\x20World!"},
		{"Hello World!", NOESCAPE_SPACE, "Hello World!"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := escaped(test.input, test.escape)
			if result != test.want {
				t.Errorf("got %q, want %q", result, test.want)
			}
		})
	}
}

func TestDumpNode(t *testing.T) {
	modTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	regularFileEntry := &internal.FileMetadata{
		Name:     "example.txt",
		Size:     100,
		Type:     internal.TypeReg,
		UID:      1000,
		GID:      1000,
		Devmajor: 0,
		Devminor: 0,
		ModTime:  &modTime,
		Linkname: "",
		Digest:   "sha256:abcdef1234567890",
		Xattrs: map[string]string{
			"user.key1": "value1",
		},
	}

	directoryEntry := &internal.FileMetadata{
		Name:     "mydir",
		Type:     internal.TypeDir,
		UID:      1000,
		GID:      1000,
		ModTime:  &modTime,
		Linkname: "",
		Xattrs: map[string]string{
			"user.key2": "value2",
		},
	}

	symlinkEntry := &internal.FileMetadata{
		Name:     "mysymlink",
		Type:     internal.TypeSymlink,
		ModTime:  &modTime,
		Linkname: "targetfile",
	}

	hardlinkEntry := &internal.FileMetadata{
		Name:     "myhardlink",
		Type:     internal.TypeLink,
		ModTime:  &modTime,
		Linkname: "existingfile",
	}

	missingParentEntry := &internal.FileMetadata{
		Name:    "foo/bar/baz/entry",
		Type:    internal.TypeReg,
		ModTime: &modTime,
	}

	var bufRegularFile, bufDirectory, bufSymlink, bufHardlink, bufMissingParent bytes.Buffer

	added := map[string]struct{}{"/": {}}

	err := dumpNode(&bufRegularFile, added, map[string]int{}, map[string]string{}, regularFileEntry)
	if err != nil {
		t.Errorf("unexpected error for regular file: %v", err)
	}

	err = dumpNode(&bufDirectory, added, map[string]int{}, map[string]string{}, directoryEntry)
	if err != nil {
		t.Errorf("unexpected error for directory: %v", err)
	}

	err = dumpNode(&bufSymlink, added, map[string]int{}, map[string]string{}, symlinkEntry)
	if err != nil {
		t.Errorf("unexpected error for symlink: %v", err)
	}

	err = dumpNode(&bufHardlink, added, map[string]int{}, map[string]string{}, hardlinkEntry)
	if err != nil {
		t.Errorf("unexpected error for hardlink: %v", err)
	}

	err = dumpNode(&bufMissingParent, added, map[string]int{}, map[string]string{}, missingParentEntry)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expectedRegularFile := "/example.txt 100 100000 1 1000 1000 0 1672531200.0 ab/cdef1234567890 - - user.key1=value1\n"
	expectedDirectory := "/mydir 0 40000 1 1000 1000 0 1672531200.0 - - - user.key2=value2\n"
	expectedSymlink := "/mysymlink 0 120000 1 0 0 0 1672531200.0 targetfile - -\n"
	expectedHardlink := "/myhardlink 0 @100000 1 0 0 0 1672531200.0 /existingfile - -\n"
	expectedActualMissingParent := "/foo 0 40755 1 0 0 0 0.0 - - -\n/foo/bar 0 40755 1 0 0 0 0.0 - - -\n/foo/bar/baz 0 40755 1 0 0 0 0.0 - - -\n/foo/bar/baz/entry 0 100000 1 0 0 0 1672531200.0 - - -\n"

	actualRegularFile := bufRegularFile.String()
	actualDirectory := bufDirectory.String()
	actualSymlink := bufSymlink.String()
	actualHardlink := bufHardlink.String()
	actualMissingParent := bufMissingParent.String()

	if actualRegularFile != expectedRegularFile {
		t.Errorf("for regular file, got %q, want %q", actualRegularFile, expectedRegularFile)
	}

	if actualDirectory != expectedDirectory {
		t.Errorf("for directory, got %q, want %q", actualDirectory, expectedDirectory)
	}

	if actualSymlink != expectedSymlink {
		t.Errorf("for symlink, got %q, want %q", actualSymlink, expectedSymlink)
	}

	if actualHardlink != expectedHardlink {
		t.Errorf("for hardlink, got %q, want %q", actualHardlink, expectedHardlink)
	}
	if actualMissingParent != expectedActualMissingParent {
		t.Errorf("for missing parent, got %q, want %q", actualMissingParent, expectedActualMissingParent)
	}
}
