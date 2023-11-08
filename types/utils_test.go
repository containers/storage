package types

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
)

func TestDefaultStoreOpts(t *testing.T) {
	storageOpts, err := defaultStoreOptionsIsolated(true, 1000, "./storage_test.conf")

	expectedPath := filepath.Join(os.Getenv("HOME"), "1000", "containers/storage")

	assert.NilError(t, err)
	assert.Equal(t, storageOpts.RunRoot, expectedPath)
	assert.Equal(t, storageOpts.GraphRoot, expectedPath)
	assert.Equal(t, storageOpts.RootlessStoragePath, expectedPath)
}

func TestStorageConfOverrideEnvironmentDefaultConfigFileRootless(t *testing.T) {
	t.Setenv("CONTAINERS_STORAGE_CONF", "default_override_test.conf")
	defaultFile, err := DefaultConfigFile(true)

	expectedPath := "default_override_test.conf"

	assert.NilError(t, err)
	assert.Equal(t, defaultFile, expectedPath)
}

func TestStorageConfOverrideEnvironmentDefaultConfigFileRoot(t *testing.T) {
	t.Setenv("CONTAINERS_STORAGE_CONF", "default_override_test.conf")
	defaultFile, err := DefaultConfigFile(false)

	expectedPath := "default_override_test.conf"

	assert.NilError(t, err)
	assert.Equal(t, defaultFile, expectedPath)
}
