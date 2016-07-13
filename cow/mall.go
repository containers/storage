package cow

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/docker/docker/daemon/graphdriver"
	_ "github.com/docker/docker/daemon/graphdriver/register"
	"github.com/docker/docker/pkg/archive"
)

var (
	LoadError     = errors.New("error loading storage metadata")
	DuplicateName = errors.New("that name is already in use")
)

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
type Mall interface {
	GetGraphRoot() string
	GetGraphDriverName() string
	GetGraphOptions() []string
	GetGraphDriver() (graphdriver.Driver, error)
	GetLayerStore() (LayerStore, error)
	GetImageStore() (ImageStore, error)
	GetContainerStore() (ContainerStore, error)

	CreateLayer(id, parent, name, mountLabel string, writeable bool) (*Layer, error)
	CreateImage(id, name, layer, metadata string) (*Image, error)
	CreateContainer(id, name, image, layer, metadata string) (*Container, error)
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
	GetLayer(id string) (*Layer, error)
	GetImage(id string) (*Image, error)
	GetContainer(id string) (*Container, error)
	Crawl(layerID string) (*Users, error)
}

// Users holds an analysis of which layers, images, and containers depend on a
// given layer, either directly or indirectly.
type Users struct {
	LayerID            string
	LayersDirect       []string
	LayersIndirect     []string
	ImagesDirect       []string
	ImagesIndirect     []string
	ContainersDirect   []string
	ContainersIndirect []string
}

type mall struct {
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
	if fd, err := syscall.Open(filepath.Join(graphRoot, "libcow.lock"), os.O_RDWR, syscall.S_IRUSR|syscall.S_IWUSR); err != nil {
		return nil, err
	} else {
		lk := syscall.Flock_t{
			Type:   syscall.F_WRLCK,
			Whence: int16(os.SEEK_SET),
			Start:  0,
			Len:    0,
			Pid:    int32(os.Getpid()),
		}
		if err = syscall.FcntlFlock(uintptr(fd), syscall.F_SETLKW, &lk); err != nil {
			return nil, err
		}
	}
	m := &mall{
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
	return nil, LoadError
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
	return nil, LoadError
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
	return nil, LoadError
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
	return nil, LoadError
}

func (m *mall) CreateLayer(id, parent, name, mountLabel string, writeable bool) (*Layer, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	return rlstore.Create(id, parent, name, mountLabel, nil, writeable)
}

func (m *mall) CreateImage(id, name, layer, metadata string) (*Image, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	ilayer, err := rlstore.Get(layer)
	if err != nil {
		return nil, err
	}
	if ilayer == nil {
		return nil, ErrLayerUnknown
	}
	layer = ilayer.ID
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	return ristore.Create(id, name, layer, metadata)
}

func (m *mall) CreateContainer(id, name, image, layer, metadata string) (*Container, error) {
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	cimage, err := ristore.Get(image)
	if err != nil {
		return nil, err
	}
	if cimage == nil {
		return nil, ErrImageUnknown
	}
	if layer == "" {
		rlstore, err := m.GetLayerStore()
		if err != nil {
			return nil, err
		}
		clayer, err := rlstore.Create("", cimage.TopLayer, "", "", nil, true)
		if err != nil {
			return nil, err
		}
		layer = clayer.ID
	}
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}
	return rcstore.Create(id, name, cimage.ID, layer, metadata)
}

func (m *mall) Exists(id string) bool {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return false
	}
	return rlstore.Exists(id)
}

func (m *mall) Delete(id string) error {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return err
	}
	return rlstore.Delete(id)
}

func (m *mall) Wipe() error {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return err
	}
	if err = rcstore.Wipe(); err != nil {
		return err
	}
	ristore, err := m.GetImageStore()
	if err != nil {
		return err
	}
	if err = ristore.Wipe(); err != nil {
		return err
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
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
	if c, err := rcstore.Get(id); c != nil && err == nil {
		id = c.LayerID
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return "", err
	}
	return rlstore.Mount(id, mountLabel)
}

func (m *mall) Unmount(id string) error {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return err
	}
	if c, err := rcstore.Get(id); c != nil && err == nil {
		id = c.LayerID
	}
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return err
	}
	return rlstore.Unmount(id)
}

func (m *mall) Changes(from, to string) ([]archive.Change, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	return rlstore.Changes(from, to)
}

func (m *mall) DiffSize(from, to string) (int64, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return -1, err
	}
	return rlstore.DiffSize(from, to)
}

func (m *mall) Diff(from, to string) (archive.Reader, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	return rlstore.Diff(from, to)
}

func (m *mall) ApplyDiff(to string, diff archive.Reader) (int64, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return -1, err
	}
	return rlstore.ApplyDiff(to, diff)
}

func (m *mall) Layers() ([]Layer, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	return rlstore.Layers()
}

func (m *mall) Images() ([]Image, error) {
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	return ristore.Images()
}

func (m *mall) Containers() ([]Container, error) {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}
	return rcstore.Containers()
}

func (m *mall) GetLayer(id string) (*Layer, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	return rlstore.Get(id)
}

func (m *mall) GetImage(id string) (*Image, error) {
	ristore, err := m.GetImageStore()
	if err != nil {
		return nil, err
	}
	return ristore.Get(id)
}

func (m *mall) GetContainer(id string) (*Container, error) {
	rcstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}
	return rcstore.Get(id)
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
	if layer, err := rlstore.Get(layerID); err == nil {
		layerID = layer.ID
	}
	u := &Users{LayerID: layerID}
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
	children := make(map[string]*[]string)
	for _, layer := range layers {
		if childs, known := children[layer.Parent]; known {
			newChildren := append(*childs, layer.ID)
			children[layer.Parent] = &newChildren
		} else {
			children[layer.Parent] = &[]string{layer.ID}
		}
	}
	if childs, known := children[layerID]; known {
		u.LayersDirect = *childs
	}
	indirects := []string{}
	examined := make(map[string]string)
	queue := u.LayersDirect
	for n := 0; n < len(queue); n++ {
		if _, skip := examined[queue[n]]; skip {
			continue
		}
		examined[queue[n]] = queue[n]
		more := children[queue[n]]
		if more != nil {
			for _, child := range *more {
				queue = append(queue, child)
				indirects = append(indirects, child)
			}
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
			if examined[image.TopLayer] == image.TopLayer {
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
			if examined[container.LayerID] == container.LayerID {
				if u.ContainersIndirect == nil {
					u.ContainersIndirect = []string{container.LayerID}
				} else {
					u.ContainersIndirect = append(u.ContainersIndirect, container.LayerID)
				}
			}
		}
	}
	return u, nil
}
