package path

import (
	"fmt"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanAbsPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"", "/"},
		{".", "/"},
		{"..", "/"},
		{"foo/../..", "/"},
		{"/foo/../..", "/"},
		{"./", "/"},
		{"../", "/"},
		{"/../", "/"},
		{"/./", "/"},
		{"foo", "/foo"},
		{"foo/bar", "/foo/bar"},
		{"/foo/bar/../baz", "/foo/baz"},
		{"/foo/./bar", "/foo/bar"},
		{"/foo/bar/../../baz", "/baz"},
		{"/././foo", "/foo"},
		{"../foo", "/foo"},
		{"./foo/bar/../..", "/"},
		{"foo/..", "/"},
		{"foo/../bar", "/bar"},
		{"//foo//bar", "/foo/bar"},
		{"foo/bar//baz/..", "/foo/bar"},
		{"../..", "/"},
		{".././..", "/"},
		{"../../.", "/"},
		{"/../../foo", "/foo"},
		{"../foo/bar/../baz", "/foo/baz"},
		{"../.././/.//../foo/./../bar/..", "/"},
		{"a/../.././/.//../foo/./../bar/..", "/"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, CleanAbsPath(test.path), fmt.Sprintf("path %q failed", test.path))
	}
}

func TestRegularFilePathForValidatedDigest(t *testing.T) {
	d, err := digest.Parse("sha256:0123456789abcdef1123456789abcdef2123456789abcdef3123456789abcdef")
	require.NoError(t, err)
	res, err := RegularFilePathForValidatedDigest(d)
	require.NoError(t, err)
	assert.Equal(t, "01/23456789abcdef1123456789abcdef2123456789abcdef3123456789abcdef", res)

	d, err = digest.Parse("sha512:0123456789abcdef1123456789abcdef2123456789abcdef3123456789abcdef0123456789abcdef1123456789abcdef2123456789abcdef3123456789abcdef")
	require.NoError(t, err)
	_, err = RegularFilePathForValidatedDigest(d)
	assert.Error(t, err)
}
