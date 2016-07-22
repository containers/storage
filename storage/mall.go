package storage

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	// register all of the built-in drivers
	_ "github.com/containers/storage/drivers/register"

	"github.com/containers/storage/drivers"
	"github.com/containers/storage/pkg/archive"
)

var (
	errLoadError     = errors.New("error loading storage metadata")
	errDuplicateName = errors.New("that name is already in use")
)

// Store wraps up the most common methods of the various types of file-based
// data stores that we implement.
//
// Load() reloads the contents of the store from disk.  It should be called
// with the lock held.
//
// Save() saves the contents of the store to disk.  It should be called with
// the lock held, and Touch() should be called afterward before releasing the
// lock.
type Store interface {
	Locker
	Load() error
	Save() error
}

// Mall wraps up the various types of stores that we use into a singleton
// object that initializes and manages them all together.
//
// GetGraphRoot, GetGraphDriverName, and GetGraphOptions retrieve settings that
// were passed to MakeMall() when the object was created.
//
// GetGraphDriver obtains and returns a handle to the graph Driver object used
// by the Mall.
//
// GetLayerStore obtains and returns a handle to the layer store object used by
// the Mall.
//
// CreateLayer creates a new layer in the underlying storage driver, optionally
// having the specified ID (one will be assigned if none is specified), with
// the specified layer (or no layer) as its parent, and with an optional name.
// (The writeable flag is ignored.)
//
// CreateImage creates a new image, optionally with the specified ID (one will
// be assigned if none is specified), with an optional name, and referring to a
// specified image and with optional metadata.  An image is a record which
// associates the ID of a layer with a caller-supplied metadata string which
// the library stores for the convenience of the caller.
//
// CreateContainer creates a new container, optionally with the specified ID
// (one will be assigned if none is specified), with an optional name, using
// the specified image's top layer as the basis for the container's layer, and
// assigning the specified ID to that layer (one will be created if none is
// specified).  A container is a layer which is associated with a metadata
// string which the library stores for the convenience of the caller.
//
// SetMetadata updates the metadata which is associated with an image or
// container (whichever the passed-in ID refers to) to match the specified
// value.  The metadata value can be retrieved at any time using GetImage or
// GetContainer.
//
// Exists checks if there is a layer, image, or container which has the
// passed-in ID or name.
//
// Status asks for a status report, in the form of key-value pairs, from the
// underlying storage driver.  The contents vary from driver to driver.
//
// Delete removes the layer, image, or container which has the passed-in ID or
// name.  Note that no safety checks are performed, so this can leave images
// with references to layers which do not exist, and layers with references to
// parents which no longer exist.
//
// Wipe removes all known layers, images, and containers.
//
// Mount attempts to mount a layer, image, or container for access, and returns
// the pathname if it succeeds.
//
// Unmount attempts to unmount a layer, image, or container, given an ID, a
// name, or a mount path.
//
// Changes returns a summary of the changes which would need to be made to one
// layer to make its contents the same as a second layer.  If the first layer
// is not specified, the second layer's parent is assumed.  Each Change
// structure contains a Path relative to the layer's root directory, and a Kind
// which is either ChangeAdd, ChangeModify, or ChangeDelete.
//
// DiffSize returns a count of the size of the tarstream which would specify
// the changes returned by Changes.
//
// Diff returns the tarstream which would specify the changes returned by
// Changes.
//
// ApplyDiff applies a tarstream to a layer.
//
// Layers returns a list of the currently known layers.
//
// Images returns a list of the currently known images.
//
// Containers returns a list of the currently known containers.
//
// GetNames returns the list of names for a layer, image, or container.
//
// SetNames changes the list of names for a layer, image, or container.
//
// GetLayer returns a specific layer.
//
// GetImage returns a specific image.
//
// GetImagesByTopLayer returns a list of images which reference the specified
// layer as their top layer.  They will have different names and may have
// different metadata.
//
// GetContainer returns a specific container.
//
// GetContainerByLayer returns a specific container based on its layer ID or
// name.
//
// Lookup returns the ID of a layer, image, or container with the specified
// name.
//
// Crawl enumerates all of the layers, images, and containers which depend on
// or refer to, either directly or indirectly, the specified layer, top layer
// of an image, or container layer.
type Mall interface {
	GetGraphRoot() string
	GetGraphDriverName() string
	GetGraphOptions() []string
	GetGraphDriver() (graphdriver.Driver, error)
	GetLayerStore() (LayerStore, error)
	GetImageStore() (ImageStore, error)
	GetContainerStore() (ContainerStore, error)

	CreateLayer(id, parent string, names []string, mountLabel string, writeable bool) (*Layer, error)
	CreateImage(id string, names []string, layer, metadata string) (*Image, error)
	CreateContainer(id string, names []string, image, layer, metadata string) (*Container, error)
	SetMetadata(id, metadata string) error
	Exists(id string) bool
	Status() ([][2]string, error)
	Delete(id string) error
	Wipe() error
	Mount(id, mountLabel string) (string, error)
	Unmount(id string) error
	Changes(from, to string) ([]archive.Change, error)
	DiffSize(from, to string) (int64, error)
	Diff(from, to string) (archive.Reader, error)
	ApplyDiff(to string, diff archive.Reader) (int64, error)
	Layers() ([]Layer, error)
	Images() ([]Image, error)
	Containers() ([]Container, error)
	GetNames(id string) ([]string, error)
	SetNames(id string, names []string) error
	GetLayer(id string) (*Layer, error)
	GetImage(id string) (*Image, error)
	GetImagesByTopLayer(id string) ([]*Image, error)
	GetContainer(id string) (*Container, error)
	GetContainerByLayer(id string) (*Container, error)
	Lookup(name string) (string, error)
	Crawl(layerID string) (*Users, error)
}

// Users holds an analysis of which layers, images, and containers depend on a
// given layer, either directly or indirectly.
type Users struct {
	ID                 string   `json:"id"`
	LayerID            string   `json:"layer"`
	LayersDirect       []string `json:"directlayers,omitempty"`
	LayersIndirect     []string `json:"indirectlayers,omitempty"`
	ImagesDirect       []string `json:"directimages,omitempty"`
	ImagesIndirect     []string `json:"indirectimages,omitempty"`
	ContainersDirect   []string `json:"directcontainers,omitempty"`
	ContainersIndirect []string `json:"indirectcontainers,omitempty"`
}

type mall struct {
	lockfile        sync.Locker
	graphRoot       string
	graphDriverName string
	graphOptions    []string
	loaded          bool
	graphDriver     graphdriver.Driver
	layerStore      LayerStore
	imageStore      ImageStore
	containerStore  ContainerStore
}

// MakeMall creates and initializes a new Mall object, and the underlying
// storage that it controls.
func MakeMall(graphRoot, graphDriverName string, graphOptions []string) (Mall, error) {
	if err := os.MkdirAll(graphRoot, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	for _, subdir := range []string{"mounts", "tmp", graphDriverName} {
		if err := os.MkdirAll(filepath.Join(graphRoot, subdir), 0700); err != nil && !os.IsExist(err) {
			return nil, err
		}
	}
	lockfile, err := GetLockfile(filepath.Join(graphRoot, "storage.lock"))
	if err != nil {
		return nil, err
	}
	m := &mall{
		lockfile:        lockfile,
		graphRoot:       graphRoot,
		graphDriverName: graphDriverName,
		graphOptions:    graphOptions,
	}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *mall) GetGraphDriverName() string {
	return m.graphDriverName
}

func (m *mall) GetGraphRoot() string {
	return m.graphRoot
}

func (m *mall) GetGraphOptions() []string {
	return m.graphOptions
}

func (m *mall) load() error {
	driver, err := graphdriver.New(m.graphRoot, m.graphDriverName, m.graphOptions, nil, nil)
	if err != nil {
		return err
	}

	rlpath := filepath.Join(m.graphRoot, "layers")
	if err := os.MkdirAll(rlpath, 0700); err != nil {
		return err
	}
	rls, err := newLayerStore(rlpath, driver)
	if err != nil {
		return err
	}
	m.layerStore = rls
	ripath := filepath.Join(m.graphRoot, "images")
	if err := os.MkdirAll(ripath, 0700); err != nil {
		return err
	}
	ris, err := newImageStore(ripath)
	if err != nil {
		return err
	}
	m.imageStore = ris
	rcpath := filepath.Join(m.graphRoot, "containers")
	if err := os.MkdirAll(rcpath, 0700); err != nil {
		return err
	}
	rcs, err := newContainerStore(rcpath)
	if err != nil {
		return err
	}
	m.containerStore = rcs

	m.loaded = true
	return nil
}

func (m *mall) GetGraphDriver() (graphdriver.Driver, error) {
	if !m.loaded {
		if err := m.load(); err != nil {
			return nil, err
		}
	}
	if m.graphDriver != nil {
		return m.graphDriver, nil
	}
	return nil, errLoadError
}

func (m *mall) GetLayerStore() (LayerStore, error) {
	if !m.loaded {
		if err := m.load(); err != nil {
			return nil, err
		}
	}
	if m.layerStore != nil {
		return m.layerStore, nil
	}
	return nil, errLoadError
}

func (m *mall) GetImageStore() (ImageStore, error) {
	if !m.loaded {
		if err := m.load(); err != nil {
			return nil, err
		}
	}
	if m.imageStore != nil {
		return m.imageStore, nil
	}
	return nil, errLoadError
}

func (m *mall) GetContainerStore() (ContainerStore, error) {
	if !m.loaded {
		if err := m.load(); err != nil {
			return nil, err
		}
	}
	if m.containerStore != nil {
		return m.containerStore, nil
	}
	return nil, errLoadError
}

func (m *mall) CreateLayer(id, parent string, names []string, mountLabel string, writeable bool) (*Layer, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	defer rlstore.Touch()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	defer ristore.Touch()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	defer rcstore.Touch()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	if id != "" {
		if l, err := rlstore.Get(id); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := ristore.Get(id); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := rcstore.Get(id); l != nil && err == nil {
			return nil, errDuplicateName
		}
	}
	for _, name := range names {
		if l, err := rlstore.Get(name); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := ristore.Get(name); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := rcstore.Get(name); l != nil && err == nil {
			return nil, errDuplicateName
		}
	}

	return rlstore.Create(id, parent, names, mountLabel, nil, writeable)
}

func (m *mall) CreateImage(id string, names []string, layer, metadata string) (*Image, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	defer ristore.Touch()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	defer rcstore.Touch()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	if id != "" {
		if l, err := rlstore.Get(id); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := ristore.Get(id); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := rcstore.Get(id); l != nil && err == nil {
			return nil, errDuplicateName
		}
	}
	for _, name := range names {
		if l, err := rlstore.Get(name); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := ristore.Get(name); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := rcstore.Get(name); l != nil && err == nil {
			return nil, errDuplicateName
		}
	}

	ilayer, err := rlstore.Get(layer)
	if err != nil {
		return nil, err
	}
	if ilayer == nil {
		return nil, ErrLayerUnknown
	}
	layer = ilayer.ID
	return ristore.Create(id, names, layer, metadata)
}

func (m *mall) CreateContainer(id string, names []string, image, layer, metadata string) (*Container, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	defer rlstore.Touch()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	defer rcstore.Touch()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if id != "" {
		if l, err := rlstore.Get(id); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := ristore.Get(id); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := rcstore.Get(id); l != nil && err == nil {
			return nil, errDuplicateName
		}
	}
	if layer != "" {
		if l, err := rlstore.Get(layer); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := ristore.Get(layer); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := rcstore.Get(layer); l != nil && err == nil {
			return nil, errDuplicateName
		}
	}
	for _, name := range names {
		if l, err := rlstore.Get(name); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := ristore.Get(name); l != nil && err == nil {
			return nil, errDuplicateName
		}
		if l, err := rcstore.Get(name); l != nil && err == nil {
			return nil, errDuplicateName
		}
	}

	cimage, err := ristore.Get(image)
	if err != nil {
		return nil, err
	}
	if cimage == nil {
		return nil, ErrImageUnknown
	}
	clayer, err := rlstore.Create(layer, cimage.TopLayer, nil, "", nil, true)
	if err != nil {
		return nil, err
	}
	layer = clayer.ID
	return rcstore.Create(id, names, cimage.ID, layer, metadata)
}

func (m *mall) SetMetadata(id, metadata string) error {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return err
	}

	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if rcstore.Exists(id) {
		defer rcstore.Touch()
		return rcstore.SetMetadata(id, metadata)
	}
	if ristore.Exists(id) {
		defer ristore.Touch()
		return ristore.SetMetadata(id, metadata)
	}
	return ErrImageUnknown
}

func (m *mall) Exists(id string) bool {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return false
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return false
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return false
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if rcstore.Exists(id) {
		return true
	}
	if ristore.Exists(id) {
		return true
	}
	return rlstore.Exists(id)
}

func (m *mall) SetNames(id string, names []string) error {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	deduped := []string{}
	seen := make(map[string]bool)
	for _, name := range names {
		if _, wasSeen := seen[name]; !wasSeen {
			seen[name] = true
			deduped = append(deduped, name)
		}
	}

	if rcstore.Exists(id) {
		return rcstore.SetNames(id, deduped)
	}
	if ristore.Exists(id) {
		return ristore.SetNames(id, deduped)
	}
	if rlstore.Exists(id) {
		return rlstore.SetNames(id, deduped)
	}
	return ErrLayerUnknown
}

func (m *mall) GetNames(id string) ([]string, error) {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if c, err := rcstore.Get(id); c != nil && err == nil {
		return c.Names, nil
	}
	if i, err := ristore.Get(id); i != nil && err == nil {
		return i.Names, nil
	}
	if l, err := rlstore.Get(id); l != nil && err == nil {
		return l.Names, nil
	}
	return nil, ErrLayerUnknown
}

func (m *mall) Lookup(name string) (string, error) {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return "", err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return "", err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return "", err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if c, err := rcstore.Get(name); c != nil && err == nil {
		return c.ID, nil
	}
	if i, err := ristore.Get(name); i != nil && err == nil {
		return i.ID, nil
	}
	if l, err := rlstore.Get(name); l != nil && err == nil {
		return l.ID, nil
	}
	return "", ErrLayerUnknown
}

func (m *mall) Delete(id string) error {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return err
	}
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if rcstore.Exists(id) {
		defer rlstore.Touch()
		defer rcstore.Touch()
		if container, err := rcstore.Get(id); err == nil {
			if err := rlstore.Delete(container.LayerID); err != nil {
				return err
			}
			return rcstore.Delete(id)
		}
	}
	if ristore.Exists(id) {
		defer ristore.Touch()
		return ristore.Delete(id)
	}
	if rlstore.Exists(id) {
		defer rlstore.Touch()
		return rlstore.Delete(id)
	}
	return ErrLayerUnknown
}

func (m *mall) Wipe() error {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	defer rlstore.Touch()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	defer ristore.Touch()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	defer rcstore.Touch()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if err = rcstore.Wipe(); err != nil {
		return err
	}
	if err = ristore.Wipe(); err != nil {
		return err
	}
	return rlstore.Wipe()
}

func (m *mall) Status() ([][2]string, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	return rlstore.Status()
}

func (m *mall) Mount(id, mountLabel string) (string, error) {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return "", err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return "", err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	defer rlstore.Touch()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if c, err := rcstore.Get(id); c != nil && err == nil {
		id = c.LayerID
	}
	return rlstore.Mount(id, mountLabel)
}

func (m *mall) Unmount(id string) error {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	defer rlstore.Touch()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if c, err := rcstore.Get(id); c != nil && err == nil {
		id = c.LayerID
	}
	return rlstore.Unmount(id)
}

func (m *mall) Changes(from, to string) ([]archive.Change, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}

	return rlstore.Changes(from, to)
}

func (m *mall) DiffSize(from, to string) (int64, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return -1, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}

	return rlstore.DiffSize(from, to)
}

func (m *mall) Diff(from, to string) (archive.Reader, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}

	return rlstore.Diff(from, to)
}

func (m *mall) ApplyDiff(to string, diff archive.Reader) (int64, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return -1, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}

	return rlstore.ApplyDiff(to, diff)
}

func (m *mall) Layers() ([]Layer, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}

	return rlstore.Layers()
}

func (m *mall) Images() ([]Image, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}

	return ristore.Images()
}

func (m *mall) Containers() ([]Container, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	return rcstore.Containers()
}

func (m *mall) GetLayer(id string) (*Layer, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}

	return rlstore.Get(id)
}

func (m *mall) GetImage(id string) (*Image, error) {
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}

	return ristore.Get(id)
}

func (m *mall) GetImagesByTopLayer(id string) ([]*Image, error) {
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}

	layer, err := rlstore.Get(id)
	if err != nil {
		return nil, err
	}
	images := []*Image{}
	imageList, err := ristore.Images()
	if err != nil {
		return nil, err
	}
	for _, image := range imageList {
		if image.TopLayer == layer.ID {
			images = append(images, &image)
		}
	}

	return images, nil
}

func (m *mall) GetContainer(id string) (*Container, error) {
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	return rcstore.Get(id)
}

func (m *mall) GetContainerByLayer(id string) (*Container, error) {
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	layer, err := rlstore.Get(id)
	if err != nil {
		return nil, err
	}
	containerList, err := rcstore.Containers()
	if err != nil {
		return nil, err
	}
	for _, container := range containerList {
		if container.LayerID == layer.ID {
			return &container, nil
		}
	}

	return nil, ErrContainerUnknown
}

func (m *mall) Crawl(layerID string) (*Users, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	u := &Users{}
	if container, err := rcstore.Get(layerID); err == nil {
		u.ID = container.ID
		layerID = container.LayerID
	}
	if image, err := ristore.Get(layerID); err == nil {
		u.ID = image.ID
		layerID = image.TopLayer
	}
	if layer, err := rlstore.Get(layerID); err == nil {
		u.ID = layer.ID
		layerID = layer.ID
	}
	if u.ID == "" {
		return nil, ErrLayerUnknown
	}
	u.LayerID = layerID
	layers, err := rlstore.Layers()
	if err != nil {
		return nil, err
	}
	images, err := ristore.Images()
	if err != nil {
		return nil, err
	}
	containers, err := rcstore.Containers()
	if err != nil {
		return nil, err
	}
	children := make(map[string][]string)
	for _, layer := range layers {
		if childs, known := children[layer.Parent]; known {
			newChildren := append(childs, layer.ID)
			children[layer.Parent] = newChildren
		} else {
			children[layer.Parent] = []string{layer.ID}
		}
	}
	if childs, known := children[layerID]; known {
		u.LayersDirect = childs
	}
	indirects := []string{}
	examined := make(map[string]bool)
	queue := u.LayersDirect
	for n := 0; n < len(queue); n++ {
		if _, skip := examined[queue[n]]; skip {
			continue
		}
		examined[queue[n]] = true
		for _, child := range children[queue[n]] {
			queue = append(queue, child)
			indirects = append(indirects, child)
		}
	}
	u.LayersIndirect = indirects
	for _, image := range images {
		if image.TopLayer == layerID {
			if u.ImagesDirect == nil {
				u.ImagesDirect = []string{image.ID}
			} else {
				u.ImagesDirect = append(u.ImagesDirect, image.ID)
			}
		} else {
			if _, isDescended := examined[image.TopLayer]; isDescended {
				if u.ImagesIndirect == nil {
					u.ImagesIndirect = []string{image.ID}
				} else {
					u.ImagesIndirect = append(u.ImagesIndirect, image.ID)
				}
			}
		}
	}
	for _, container := range containers {
		if container.LayerID == layerID {
			if u.ContainersDirect == nil {
				u.ContainersDirect = []string{container.ID}
			} else {
				u.ContainersDirect = append(u.ContainersDirect, container.ID)
			}
		} else {
			if _, isDescended := examined[container.LayerID]; isDescended {
				if u.ContainersIndirect == nil {
					u.ContainersIndirect = []string{container.ID}
				} else {
					u.ContainersIndirect = append(u.ContainersIndirect, container.ID)
				}
			}
		}
	}
	return u, nil
}
