package path

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
