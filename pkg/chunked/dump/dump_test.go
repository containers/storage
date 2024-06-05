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
		{"NetLock_Arany_=Class_Gold=_Főtanúsítvány.crt", 0, `NetLock_Arany_=Class_Gold=_F\xc5\x91tan\xc3\xbas\xc3\xadtv\xc3\xa1ny.crt`},
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

	rootEntry := &internal.FileMetadata{
		Name:    "./",
		Type:    internal.TypeDir,
		ModTime: &modTime,
		UID:     0,
	}

	rootEntry2 := &internal.FileMetadata{
		Name:    "./",
		Type:    internal.TypeDir,
		ModTime: &modTime,
		UID:     101,
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

	testCases := []struct {
		name                string
		entries             []*internal.FileMetadata
		expected            string
		skipAddingRootEntry bool
		expectError         bool
	}{
		{
			name: "root entry",
			entries: []*internal.FileMetadata{
				rootEntry,
			},
			expected:            "/ 0 40000 1 0 0 0 1672531200.0 - - -\n",
			skipAddingRootEntry: true,
		},
		{
			name: "duplicate root entry",
			entries: []*internal.FileMetadata{
				rootEntry,
				rootEntry,
				rootEntry,
			},
			expected:            "/ 0 40000 1 0 0 0 1672531200.0 - - -\n",
			skipAddingRootEntry: true,
		},
		{
			name: "duplicate root entry with mismatch",
			entries: []*internal.FileMetadata{
				rootEntry,
				rootEntry2,
			},
			skipAddingRootEntry: true,
			expectError:         true,
		},
		{
			name: "regular file",
			entries: []*internal.FileMetadata{
				regularFileEntry,
			},
			expected: "/example.txt 100 100000 1 1000 1000 0 1672531200.0 ab/cdef1234567890 - - user.key1=value1\n",
		},
		{
			name: "root entry with file",
			entries: []*internal.FileMetadata{
				rootEntry,
				regularFileEntry,
			},
			expected:            "/ 0 40000 1 0 0 0 1672531200.0 - - -\n/example.txt 100 100000 1 1000 1000 0 1672531200.0 ab/cdef1234567890 - - user.key1=value1\n",
			skipAddingRootEntry: true,
		},
		{
			name: "directory",
			entries: []*internal.FileMetadata{
				directoryEntry,
			},
			expected: "/mydir 0 40000 1 1000 1000 0 1672531200.0 - - - user.key2=value2\n",
		},
		{
			name: "symlink",
			entries: []*internal.FileMetadata{
				symlinkEntry,
			},
			expected: "/mysymlink 0 120000 1 0 0 0 1672531200.0 targetfile - -\n",
		},
		{
			name: "hardlink",
			entries: []*internal.FileMetadata{
				hardlinkEntry,
			},
			expected: "/myhardlink 0 @100000 1 0 0 0 1672531200.0 /existingfile - -\n",
		},
		{
			name: "missing parent",
			entries: []*internal.FileMetadata{
				missingParentEntry,
			},
			expected: "/foo 0 40755 1 0 0 0 0.0 - - -\n/foo/bar 0 40755 1 0 0 0 0.0 - - -\n/foo/bar/baz 0 40755 1 0 0 0 0.0 - - -\n/foo/bar/baz/entry 0 100000 1 0 0 0 1672531200.0 - - -\n",
		},
		{
			name: "complex case",
			entries: []*internal.FileMetadata{
				rootEntry,
				regularFileEntry,
				directoryEntry,
			},
			expected:            "/ 0 40000 1 0 0 0 1672531200.0 - - -\n/example.txt 100 100000 1 1000 1000 0 1672531200.0 ab/cdef1234567890 - - user.key1=value1\n/mydir 0 40000 1 1000 1000 0 1672531200.0 - - - user.key2=value2\n",
			skipAddingRootEntry: true,
		},
	}

	for _, testCase := range testCases {
		var buf bytes.Buffer

		added := map[string]*internal.FileMetadata{}
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
