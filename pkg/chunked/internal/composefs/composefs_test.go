package composefs

import (
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
