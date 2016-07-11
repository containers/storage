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
	LoadError        = errors.New("error loading storage metadata")
	InvalidImageName = errors.New("invalid name for new image")
)

type Mall interface {
	GetGraphDriverName() string
	GetGraphDriver() (graphdriver.Driver, error)
	GetLayerStore() (LayerStore, error)

	Create(id, parent, name, mountLabel string, writeable bool) (*Layer, error)
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
}

type mall struct {
	graphRoot       string
	graphDriverName string
	graphOptions    []string
	loaded          bool
	graphDriver     graphdriver.Driver
	LayerStore      LayerStore
}

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
	m.LayerStore = rls

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
	if m.LayerStore != nil {
		return m.LayerStore, nil
	}
	return nil, LoadError
}

func (m *mall) Create(id, parent, name, mountLabel string, writeable bool) (*Layer, error) {
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return nil, err
	}
	return rlstore.Create(id, parent, name, mountLabel, nil, writeable)
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
	rlstore, err := m.GetLayerStore()
	if err != nil {
		return "", err
	}
	return rlstore.Mount(id, mountLabel)
}

func (m *mall) Unmount(id string) error {
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
