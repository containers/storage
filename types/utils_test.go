package types

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/storage/pkg/unshare"
	"gotest.tools/v3/assert"
)

func TestDefaultStoreOpts(t *testing.T) {
	if !usePerUserStorage() {
		t.Skip()
	}
	storageOpts, err := loadStoreOptionsFromConfFile("./storage_test.conf")
	expectedPath := filepath.Join(os.Getenv("HOME"), fmt.Sprintf("%d", unshare.GetRootlessUID()), "containers/storage")

	assert.NilError(t, err)
	assert.Equal(t, storageOpts.RunRoot, expectedPath)
	assert.Equal(t, storageOpts.GraphRoot, expectedPath)
	assert.Equal(t, storageOpts.RootlessStoragePath, expectedPath)
}

func TestStorageConfOverrideEnvironmentDefaultConfigFileRootless(t *testing.T) {
	t.Setenv("CONTAINERS_STORAGE_CONF", "default_override_test.conf")
	defaultFile, err := DefaultConfigFile()

	expectedPath := "default_override_test.conf"

	assert.NilError(t, err)
	assert.Equal(t, defaultFile, expectedPath)
}

func TestStorageConfOverrideEnvironmentDefaultConfigFileRoot(t *testing.T) {
	t.Setenv("CONTAINERS_STORAGE_CONF", "default_override_test.conf")
	defaultFile, err := DefaultConfigFile()

	expectedPath := "default_override_test.conf"

	assert.NilError(t, err)
	assert.Equal(t, defaultFile, expectedPath)
}
