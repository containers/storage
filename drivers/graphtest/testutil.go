//go:build linux || freebsd

// ^^ The code is conceptually portable, but only called from within *_unix.go in this package.
// So it is excluded to avoid warnings on other platforms.

package graphtest

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"

	graphdriver "github.com/containers/storage/drivers"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/stringid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func randomContent(size int, seed int64) []byte {
	content := make([]byte, size)

	rng := rand.New(rand.NewSource(seed))
	read, err := rng.Read(content)
	if err != nil || read != size {
		panic("Unexpected failure of math/rand.Rand.Read")
	}

	return content
}

func addFiles(drv graphdriver.Driver, layer string, seed int64) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	if err := os.WriteFile(path.Join(root, "file-a"), randomContent(64, seed), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Join(root, "dir-b"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path.Join(root, "dir-b", "file-b"), randomContent(128, seed+1), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path.Join(root, "file-c"), randomContent(128*128, seed+2), 0o755)
}

func checkFile(drv graphdriver.Driver, layer, filename string, content []byte) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	fileContent, err := os.ReadFile(path.Join(root, filename))
	if err != nil {
		return err
	}

	if !bytes.Equal(fileContent, content) {
		return fmt.Errorf("mismatched file content %v, expecting %v", fileContent, content)
	}

	return nil
}

func addFile(drv graphdriver.Driver, layer, filename string, content []byte) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	return os.WriteFile(path.Join(root, filename), content, 0o755)
}

func addDirectory(drv graphdriver.Driver, layer, dir string) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	return os.MkdirAll(path.Join(root, dir), 0o755)
}

func removeAll(drv graphdriver.Driver, layer string, names ...string) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	for _, filename := range names {
		if err := os.RemoveAll(path.Join(root, filename)); err != nil {
			return err
		}
	}
	return nil
}

func checkFileRemoved(drv graphdriver.Driver, layer, filename string) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	if _, err := os.Stat(path.Join(root, filename)); err == nil {
		return fmt.Errorf("file still exists: %s", path.Join(root, filename))
	} else if !os.IsNotExist(err) {
		return err
	}

	return nil
}

func addManyFiles(drv graphdriver.Driver, layer string, count int, seed int64) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	for i := 0; i < count; i += 100 {
		dir := path.Join(root, fmt.Sprintf("directory-%d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		for j := 0; i+j < count && j < 100; j++ {
			file := path.Join(dir, fmt.Sprintf("file-%d", i+j))
			if err := os.WriteFile(file, randomContent(64, seed+int64(i+j)), 0o755); err != nil {
				return err
			}
		}
	}

	return nil
}

func changeManyFiles(drv graphdriver.Driver, layer string, count int, seed int64) ([]archive.Change, error) {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	changes := []archive.Change{}
	for i := 0; i < count; i += 100 {
		archiveRoot := fmt.Sprintf("/directory-%d", i)
		if err := os.MkdirAll(path.Join(root, archiveRoot), 0o755); err != nil {
			return nil, err
		}
		for j := 0; i+j < count && j < 100; j++ {
			if j == 0 {
				changes = append(changes, archive.Change{
					Path: archiveRoot,
					Kind: archive.ChangeModify,
				})
			}
			var change archive.Change
			switch j % 3 {
			// Update file
			case 0:
				var originalFileInfo, updatedFileInfo os.FileInfo
				change.Path = path.Join(archiveRoot, fmt.Sprintf("file-%d", i+j))
				change.Kind = archive.ChangeModify
				if originalFileInfo, err = os.Stat(path.Join(root, change.Path)); err != nil {
					return nil, err
				}
				for updatedFileInfo == nil || updatedFileInfo.ModTime().Equal(originalFileInfo.ModTime()) {
					if err := os.WriteFile(path.Join(root, change.Path), randomContent(64, seed+int64(i+j)), 0o755); err != nil {
						return nil, err
					}
					if updatedFileInfo, err = os.Stat(path.Join(root, change.Path)); err != nil {
						return nil, err
					}
				}
			// Add file
			case 1:
				change.Path = path.Join(archiveRoot, fmt.Sprintf("file-%d-%d", seed, i+j))
				change.Kind = archive.ChangeAdd
				if err := os.WriteFile(path.Join(root, change.Path), randomContent(64, seed+int64(i+j)), 0o755); err != nil {
					return nil, err
				}
			// Remove file
			case 2:
				change.Path = path.Join(archiveRoot, fmt.Sprintf("file-%d", i+j))
				change.Kind = archive.ChangeDelete
				if err := os.Remove(path.Join(root, change.Path)); err != nil {
					return nil, err
				}
			}
			changes = append(changes, change)
		}
	}

	return changes, nil
}

func checkManyFiles(drv graphdriver.Driver, layer string, count int, seed int64) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	for i := 0; i < count; i += 100 {
		dir := path.Join(root, fmt.Sprintf("directory-%d", i))
		for j := 0; i+j < count && j < 100; j++ {
			file := path.Join(dir, fmt.Sprintf("file-%d", i+j))
			fileContent, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			content := randomContent(64, seed+int64(i+j))

			if !bytes.Equal(fileContent, content) {
				return fmt.Errorf("mismatched file content %v, expecting %v", fileContent, content)
			}
		}
	}

	return nil
}

func addLayerFiles(drv graphdriver.Driver, layer, parent string, i int) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	if err := os.WriteFile(path.Join(root, "top-id"), []byte(layer), 0o755); err != nil {
		return err
	}
	layerDir := path.Join(root, fmt.Sprintf("layer-%d", i))
	if err := os.MkdirAll(layerDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path.Join(layerDir, "layer-id"), []byte(layer), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path.Join(layerDir, "parent-id"), []byte(parent), 0o755); err != nil {
		return err
	}

	return nil
}

func addManyLayers(t testing.TB, drv graphdriver.Driver, baseLayer string, count int) (string, error) {
	lastLayer := baseLayer
	for i := 1; i <= count; i++ {
		nextLayer := stringid.GenerateRandomID()
		if err := drv.Create(nextLayer, lastLayer, nil); err != nil {
			return "", err
		}
		t.Cleanup(func() { removeLayer(t, drv, nextLayer) })
		if err := addLayerFiles(drv, nextLayer, lastLayer, i); err != nil {
			return "", err
		}

		lastLayer = nextLayer

	}
	return lastLayer, nil
}

func checkManyLayers(drv graphdriver.Driver, layer string, count int) error {
	root, err := drv.Get(layer, graphdriver.MountOpts{})
	if err != nil {
		return err
	}
	defer func() {
		if err := drv.Put(layer); err != nil {
			logrus.Warn(err)
		}
	}()

	layerIDBytes, err := os.ReadFile(path.Join(root, "top-id"))
	if err != nil {
		return err
	}

	if !bytes.Equal(layerIDBytes, []byte(layer)) {
		return fmt.Errorf("mismatched file content %v, expecting %v", layerIDBytes, []byte(layer))
	}

	for i := count; i > 0; i-- {
		layerDir := path.Join(root, fmt.Sprintf("layer-%d", i))

		thisLayerIDBytes, err := os.ReadFile(path.Join(layerDir, "layer-id"))
		if err != nil {
			return err
		}
		if !bytes.Equal(thisLayerIDBytes, layerIDBytes) {
			return fmt.Errorf("mismatched file content %v, expecting %v", thisLayerIDBytes, layerIDBytes)
		}
		layerIDBytes, err = os.ReadFile(path.Join(layerDir, "parent-id"))
		if err != nil {
			return err
		}
	}
	return nil
}

// readDir reads a directory just like os.ReadDir()
// then hides specific files (currently "lost+found")
// so the tests don't "see" it
func readDir(dir string) ([]os.DirEntry, error) {
	a, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	b := a[:0]
	for _, x := range a {
		if x.Name() != "lost+found" { // ext4 always have this dir
			b = append(b, x)
		}
	}

	return b, nil
}

// removeLayer tries to remove the layer
func removeLayer(t testing.TB, driver graphdriver.Driver, name string) {
	err := driver.Remove(name)
	require.NoError(t, err)
}
