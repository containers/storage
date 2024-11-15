//go:build unix

package dump

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	"github.com/containers/storage/pkg/chunked/internal/minimal"
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
		{"NetLock_Arany_=Class_Gold=_Főtanúsítvány.crt", 0, `NetLock_Arany_=Class_Gold=_F\xc5\x91tan\xc3\xbas\xc3\xadtv\xc3\xa1ny.crt`},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := escaped([]byte(test.input), test.escape)
			if result != test.want {
				t.Errorf("got %q, want %q", result, test.want)
			}
		})
	}
}

func TestDumpNode(t *testing.T) {
	modTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	regularFileEntry := &minimal.FileMetadata{
		Name:     "example.txt",
		Size:     100,
		Type:     minimal.TypeReg,
		UID:      1000,
		GID:      1000,
		Devmajor: 0,
		Devminor: 0,
		ModTime:  &modTime,
		Linkname: "",
		Digest:   "sha256:0123456789abcdef1123456789abcdef2123456789abcdef3123456789abcdef",
		Xattrs: map[string]string{
			"user.key1": base64.StdEncoding.EncodeToString([]byte("value1")),
		},
	}

	rootEntry := &minimal.FileMetadata{
		Name:    "./",
		Type:    minimal.TypeDir,
		ModTime: &modTime,
		UID:     0,
	}

	rootEntry2 := &minimal.FileMetadata{
		Name:    "./",
		Type:    minimal.TypeDir,
		ModTime: &modTime,
		UID:     101,
	}

	directoryEntry := &minimal.FileMetadata{
		Name:     "mydir",
		Type:     minimal.TypeDir,
		UID:      1000,
		GID:      1000,
		ModTime:  &modTime,
		Linkname: "",
		Xattrs: map[string]string{
			"user.key2": base64.StdEncoding.EncodeToString([]byte("value2")),
		},
	}

	symlinkEntry := &minimal.FileMetadata{
		Name:     "mysymlink",
		Type:     minimal.TypeSymlink,
		ModTime:  &modTime,
		Linkname: "targetfile",
	}

	hardlinkEntry := &minimal.FileMetadata{
		Name:     "myhardlink",
		Type:     minimal.TypeLink,
		ModTime:  &modTime,
		Linkname: "existingfile",
	}

	missingParentEntry := &minimal.FileMetadata{
		Name:    "foo/bar/baz/entry",
		Type:    minimal.TypeReg,
		ModTime: &modTime,
	}

	testCases := []struct {
		name                string
		entries             []*minimal.FileMetadata
		expected            string
		skipAddingRootEntry bool
		expectError         bool
	}{
		{
			name: "root entry",
			entries: []*minimal.FileMetadata{
				rootEntry,
			},
			expected:            "/ 0 40000 1 0 0 0 1672531200.0 - - -\n",
			skipAddingRootEntry: true,
		},
		{
			name: "duplicate root entry",
			entries: []*minimal.FileMetadata{
				rootEntry,
				rootEntry,
				rootEntry,
			},
			expected:            "/ 0 40000 1 0 0 0 1672531200.0 - - -\n",
			skipAddingRootEntry: true,
		},
		{
			name: "duplicate root entry with mismatch",
			entries: []*minimal.FileMetadata{
				rootEntry,
				rootEntry2,
			},
			skipAddingRootEntry: true,
			expectError:         true,
		},
		{
			name: "regular file",
			entries: []*minimal.FileMetadata{
				regularFileEntry,
			},
			expected: "/example.txt 100 100000 1 1000 1000 0 1672531200.0 01/23456789abcdef1123456789abcdef2123456789abcdef3123456789abcdef - - user.key1=value1\n",
		},
		{
			name: "root entry with file",
			entries: []*minimal.FileMetadata{
				rootEntry,
				regularFileEntry,
			},
			expected:            "/ 0 40000 1 0 0 0 1672531200.0 - - -\n/example.txt 100 100000 1 1000 1000 0 1672531200.0 01/23456789abcdef1123456789abcdef2123456789abcdef3123456789abcdef - - user.key1=value1\n",
			skipAddingRootEntry: true,
		},
		{
			name: "directory",
			entries: []*minimal.FileMetadata{
				directoryEntry,
			},
			expected: "/mydir 0 40000 1 1000 1000 0 1672531200.0 - - - user.key2=value2\n",
		},
		{
			name: "symlink",
			entries: []*minimal.FileMetadata{
				symlinkEntry,
			},
			expected: "/mysymlink 0 120000 1 0 0 0 1672531200.0 targetfile - -\n",
		},
		{
			name: "hardlink",
			entries: []*minimal.FileMetadata{
				hardlinkEntry,
			},
			expected: "/myhardlink 0 @100000 1 0 0 0 1672531200.0 /existingfile - -\n",
		},
		{
			name: "missing parent",
			entries: []*minimal.FileMetadata{
				missingParentEntry,
			},
			expected: "/foo 0 40755 1 0 0 0 0.0 - - -\n/foo/bar 0 40755 1 0 0 0 0.0 - - -\n/foo/bar/baz 0 40755 1 0 0 0 0.0 - - -\n/foo/bar/baz/entry 0 100000 1 0 0 0 1672531200.0 - - -\n",
		},
		{
			name: "complex case",
			entries: []*minimal.FileMetadata{
				rootEntry,
				regularFileEntry,
				directoryEntry,
			},
			expected:            "/ 0 40000 1 0 0 0 1672531200.0 - - -\n/example.txt 100 100000 1 1000 1000 0 1672531200.0 01/23456789abcdef1123456789abcdef2123456789abcdef3123456789abcdef - - user.key1=value1\n/mydir 0 40000 1 1000 1000 0 1672531200.0 - - - user.key2=value2\n",
			skipAddingRootEntry: true,
		},
	}

	for _, testCase := range testCases {
		var buf bytes.Buffer

		added := map[string]*minimal.FileMetadata{}
		if !testCase.skipAddingRootEntry {
			added["/"] = rootEntry
		}
		var foundErr error
		for _, entry := range testCase.entries {
			err := dumpNode(&buf, added, map[string]int{}, map[string]string{}, entry)
			if err != nil {
				foundErr = err
			}
		}
		if testCase.expectError && foundErr == nil {
			t.Errorf("expected error for %s, got nil", testCase.name)
		}
		if !testCase.expectError {
			if foundErr != nil {
				t.Errorf("unexpected error for %s: %v", testCase.name, foundErr)
			}
			actual := buf.String()
			if actual != testCase.expected {
				t.Errorf("for %s, got %q, want %q", testCase.name, actual, testCase.expected)
			}
		}
	}
}
