package storage

import (
	"testing"

	"gotest.tools/assert"
)

func TestValidStoragePathFormat(t *testing.T) {
	// Given
	expectErr := "Unrecognized environment variable"
	invalidPaths := []struct {
		path   string
		expect string
	}{
		{"$", expectErr},
		{"$HOMEDIR", expectErr},
		{"$HOMEdir", expectErr},
		{"/test/$HOMEDIR/$USERNAME/$UID", expectErr},
		{"/test/$HOME/$USERNAME/$UID", expectErr},
		{"/test/$HOME/$USER/$UIDNUM", expectErr},
	}
	validPaths := []string{
		"$HOME",
		"$HOME/",
		"/test/path",
		"/test/$HOME",
		"/test/$HOME/path",
		"/test/$HOME/$USER/$UID",
		"/test/$HOME/$USER/$UID/path",
		"$HOME/$USER/$UID/path",
	}

	// Then
	for _, conf := range invalidPaths {
		err := validRootlessStoragePathFormat(conf.path)
		assert.Error(t, err, "Unrecognized environment variable")
	}

	for _, path := range validPaths {
		err := validRootlessStoragePathFormat(path)
		assert.NilError(t, err)
	}
}
