package idtools

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToHost(t *testing.T) {
	idMappings := []IDMap{
		{
			ContainerID: 0,
			HostID:      1000,
			Size:        1,
		},
		{
			ContainerID: 1,
			HostID:      100000,
			Size:        65536,
		},
	}

	mappings := IDMappings{
		uids: idMappings,
		gids: idMappings,
	}

	pair, err := mappings.ToHost(IDPair{UID: 0, GID: 0})
	if err != nil {
		t.Fatal(err)
	}
	if pair.UID != 1000 {
		t.Fatalf("Converted to the wrong UID")
	}
	if pair.GID != 1000 {
		t.Fatalf("Converted to the wrong GID")
	}

	pair, err = mappings.ToHost(IDPair{UID: 1000, GID: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if pair.UID != 100999 {
		t.Fatalf("Converted to the wrong UID")
	}
	if pair.GID != 100999 {
		t.Fatalf("Converted to the wrong GID")
	}
}

func TestToHostOverflow(t *testing.T) {
	idMappings := []IDMap{
		{
			ContainerID: 0,
			HostID:      1000,
			Size:        1,
		},
		{
			ContainerID: 1,
			HostID:      100000,
			Size:        65536,
		},
	}

	mappings := IDMappings{
		uids: idMappings,
		gids: idMappings,
	}

	pair, err := mappings.ToHostOverflow(IDPair{UID: 65538, GID: 0})
	if err != nil {
		t.Fatal(err)
	}
	if pair.UID != getOverflowUID() {
		t.Fatalf("Converted to the wrong UID")
	}
	if pair.GID != 1000 {
		t.Fatalf("Converted to the wrong GID")
	}

	pair, err = mappings.ToHostOverflow(IDPair{UID: 10, GID: 65539})
	if err != nil {
		t.Fatal(err)
	}
	if pair.UID != 100009 {
		t.Fatalf("Converted to the wrong UID")
	}
	if pair.GID != getOverflowGID() {
		t.Fatalf("Converted to the wrong GID")
	}
}

func TestGetRootUIDGID(t *testing.T) {
	mappingsUIDs := []IDMap{
		{
			ContainerID: 0,
			HostID:      1000,
			Size:        100,
		},
	}
	mappingsGIDs := []IDMap{
		{
			ContainerID: 0,
			HostID:      2000,
			Size:        100,
		},
	}
	uid, gid, err := GetRootUIDGID(mappingsUIDs, mappingsGIDs)
	if err != nil {
		t.Fatal(err)
	}
	if uid != 1000 {
		t.Fatalf("Detected wrong root uid in the host")
	}
	if gid != 2000 {
		t.Fatalf("Detected wrong root uid in the host")
	}

	mappingsUIDs = []IDMap{
		{
			ContainerID: 100,
			HostID:      1001,
			Size:        1,
		},
	}
	mappingsGIDs = []IDMap{
		{
			ContainerID: 200,
			HostID:      2002,
			Size:        1,
		},
	}
	uid, gid, err = GetRootUIDGID(mappingsUIDs, mappingsGIDs)
	if err != nil {
		t.Fatal(err)
	}
	if uid != 1001 {
		t.Fatalf("Detected wrong root uid in the host")
	}
	if gid != 2002 {
		t.Fatalf("Detected wrong root uid in the host")
	}

	mappingsUIDs = []IDMap{
		{
			ContainerID: 100,
			HostID:      1001,
			Size:        100,
		},
	}
	mappingsGIDs = []IDMap{
		{
			ContainerID: 200,
			HostID:      2002,
			Size:        100,
		},
	}
	_, _, err = GetRootUIDGID(mappingsUIDs, mappingsGIDs)
	if err == nil {
		t.Fatalf("Detected root user")
	}

	mappingsUIDs = []IDMap{
		{
			ContainerID: 100,
			HostID:      1001,
			Size:        1,
		},
		{
			ContainerID: 200,
			HostID:      2001,
			Size:        1,
		},
	}
	mappingsGIDs = []IDMap{
		{
			ContainerID: 100,
			HostID:      1001,
			Size:        1,
		},
		{
			ContainerID: 200,
			HostID:      2001,
			Size:        1,
		},
	}
	_, _, err = GetRootUIDGID(mappingsUIDs, mappingsGIDs)
	if err == nil {
		t.Fatalf("Detected root user")
	}
}

func TestIsContiguous(t *testing.T) {
	mappings := []IDMap{
		{
			ContainerID: 0,
			HostID:      0,
			Size:        100,
		},
		{
			ContainerID: 100,
			HostID:      100,
			Size:        100,
		},
	}
	if !IsContiguous(mappings) {
		t.Errorf("mappings %v expected to be contiguous", mappings)
	}
	mappings = []IDMap{
		{
			ContainerID: 0,
			HostID:      10000,
			Size:        100,
		},
		{
			ContainerID: 100,
			HostID:      100,
			Size:        100,
		},
	}
	if IsContiguous(mappings) {
		t.Errorf("mappings %v expected to not be contiguous", mappings)
	}

	mappings = []IDMap{
		{
			ContainerID: 10000,
			HostID:      0,
			Size:        100,
		},
		{
			ContainerID: 100,
			HostID:      100,
			Size:        100,
		},
	}
	if IsContiguous(mappings) {
		t.Errorf("mappings %v expected to not be contiguous", mappings)
	}

	mappings = []IDMap{
		{
			ContainerID: 0,
			HostID:      10,
			Size:        10,
		},
		{
			ContainerID: 10,
			HostID:      20,
			Size:        10,
		},
		{
			ContainerID: 20,
			HostID:      30,
			Size:        10,
		},
		{
			ContainerID: 30,
			HostID:      40,
			Size:        10,
		},
	}
	if !IsContiguous(mappings) {
		t.Errorf("mappings %v expected to be contiguous", mappings)
	}

	mappings = []IDMap{
		{
			ContainerID: 0,
			HostID:      10,
			Size:        10,
		},
	}
	if !IsContiguous(mappings) {
		t.Errorf("mappings %v expected to be contiguous", mappings)
	}
}

func TestParseOverrideXattr(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		expectErr     bool
		expectedUID   int
		expectedGID   int
		expectedMode  os.FileMode
		expectedMajor int
		expectedMinor int
	}{
		{
			name:         "valid regular file",
			input:        []byte("1000:1001:0644:file"),
			expectErr:    false,
			expectedUID:  1000,
			expectedGID:  1001,
			expectedMode: 0o644,
		},
		{
			name:         "valid directory",
			input:        []byte("1000:1001:0755:dir"),
			expectErr:    false,
			expectedUID:  1000,
			expectedGID:  1001,
			expectedMode: 0o755 | os.ModeDir,
		},
		{
			name:      "Invalid format: missing fields",
			input:     []byte("1000:1001"),
			expectErr: true,
		},
		{
			name:      "Invalid UID format",
			input:     []byte("invalid:1001:0644:file"),
			expectErr: true,
		},
		{
			name:         "valid symlink",
			input:        []byte("1000:1001:0777:symlink"),
			expectErr:    false,
			expectedUID:  1000,
			expectedGID:  1001,
			expectedMode: 0o777 | os.ModeSymlink,
		},
		{
			name:         "valid pipe",
			input:        []byte("1000:1001:0666:pipe"),
			expectErr:    false,
			expectedUID:  1000,
			expectedGID:  1001,
			expectedMode: 0o666 | os.ModeNamedPipe,
		},
		{
			name:          "valid block device",
			input:         []byte("1000:1001:0600:block-5-6"),
			expectErr:     false,
			expectedUID:   1000,
			expectedGID:   1001,
			expectedMode:  0o600 | os.ModeDevice,
			expectedMajor: 5,
			expectedMinor: 6,
		},
		{
			name:          "valid char device",
			input:         []byte("1000:1001:0600:char-10-11"),
			expectErr:     false,
			expectedUID:   1000,
			expectedGID:   1001,
			expectedMode:  0o600 | os.ModeCharDevice | os.ModeDevice,
			expectedMajor: 10,
			expectedMinor: 11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stat, err := parseOverrideXattr(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stat)
				assert.Equal(t, tt.expectedUID, stat.IDs.UID)
				assert.Equal(t, tt.expectedGID, stat.IDs.GID)
				assert.Equal(t, tt.expectedMode, stat.Mode)
				assert.Equal(t, tt.expectedMajor, stat.Major)
				assert.Equal(t, tt.expectedMinor, stat.Minor)
			}
		})
	}
}

func TestFormatContainersOverrideXattrDevice(t *testing.T) {
	tests := []struct {
		name     string
		uid      int
		gid      int
		mode     fs.FileMode
		major    int
		minor    int
		expected string
	}{
		{"regular file", 1000, 1011, fs.FileMode(0o644), 0, 0, "1000:1011:0644:file"},
		{"directory", 1001, 1012, fs.FileMode(0o755) | os.ModeDir, 0, 0, "1001:1012:0755:dir"},
		{"symlink", 1002, 1013, fs.FileMode(0o777) | os.ModeSymlink, 0, 0, "1002:1013:0777:symlink"},
		{"pipe", 1003, 1014, fs.FileMode(0o666) | os.ModeNamedPipe, 0, 0, "1003:1014:0666:pipe"},
		{"socket", 1004, 1015, fs.FileMode(0o700) | os.ModeSocket, 0, 0, "1004:1015:0700:socket"},
		{"block device", 1005, 1016, fs.FileMode(0o600) | os.ModeDevice, 8, 0, "1005:1016:0600:block-8-0"},
		{"char device", 1006, 1017, fs.FileMode(0o600) | os.ModeDevice | os.ModeCharDevice, 1, 3, "1006:1017:0600:char-1-3"},

		{"setuid bit", 1012, 1022, fs.FileMode(0o755) | fs.ModeSetuid, 0, 0, "1012:1022:4755:file"},
		{"setgid bit", 1013, 1023, fs.FileMode(0o755) | fs.ModeSetgid, 0, 0, "1013:1023:2755:file"},
		{"sticky bit", 1014, 1024, fs.FileMode(0o777) | fs.ModeSticky, 0, 0, "1014:1024:1777:file"},

		{"user read", 1020, 1030, fs.FileMode(0o400), 0, 0, "1020:1030:0400:file"},
		{"user write", 1021, 1031, fs.FileMode(0o200), 0, 0, "1021:1031:0200:file"},
		{"user execute", 1022, 1032, fs.FileMode(0o100), 0, 0, "1022:1032:0100:file"},

		{"group read", 1030, 1040, fs.FileMode(0o040), 0, 0, "1030:1040:0040:file"},
		{"group write", 1031, 1041, fs.FileMode(0o020), 0, 0, "1031:1041:0020:file"},
		{"group execute", 1032, 1042, fs.FileMode(0o010), 0, 0, "1032:1042:0010:file"},

		{"others read", 1040, 1050, fs.FileMode(0o004), 0, 0, "1040:1050:0004:file"},
		{"others write", 1041, 1051, fs.FileMode(0o002), 0, 0, "1041:1051:0002:file"},
		{"others execute", 1042, 1052, fs.FileMode(0o001), 0, 0, "1042:1052:0001:file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatContainersOverrideXattrDevice(tt.uid, tt.gid, tt.mode, tt.major, tt.minor)
			require.Equal(t, tt.expected, result)

			stat, err := parseOverrideXattr([]byte(result))
			require.NoError(t, err)
			require.NotNil(t, stat)
			require.Equal(t, tt.uid, stat.IDs.UID)
			require.Equal(t, tt.gid, stat.IDs.GID)
			require.Equal(t, tt.mode, stat.Mode)
			require.Equal(t, tt.major, stat.Major)
			require.Equal(t, tt.minor, stat.Minor)
		})
	}
}

func TestParseDevice(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMajor int
		wantMinor int
		expectErr bool
	}{
		{"valid block device", "block-8-1", 8, 1, false},
		{"valid char device", "char-1-3", 1, 3, false},
		{"invalid prefix", "disk-1-2", 0, 0, true},
		{"invalid format: missing dashes", "block800", 0, 0, true},
		{"valid format: extra parts", "block-3-4-option_from_the_future", 3, 4, false},
		{"invalid major number", "block-abc-0", 0, 0, true},
		{"invalid minor number", "char-1-xyz", 0, 0, true},
		{"empty string", "", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, err := parseDevice(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMajor, major)
				assert.Equal(t, tt.wantMinor, minor)
			}
		})
	}
}
