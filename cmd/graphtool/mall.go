package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/container"
	_ "github.com/docker/docker/daemon/graphdriver/register"
	"github.com/docker/docker/image"
	"github.com/docker/docker/image/tarexport"
	"github.com/docker/docker/layer"
	"github.com/docker/docker/reference"
)

var (
	LoadError = errors.New("error loading storage metadata")
)

type Mall interface {
	GetGraphDriver() string
	GetLayerStore() (layer.Store, error)
	GetImageStore() (image.Store, error)
	GetReferenceStore() (reference.Store, error)
	GetContainerStore() (container.Store, error)
	GetPetStore() (PetStore, error)
	GetImageExporter(func(string, string, string)) (image.Exporter, error)
}

type mall struct {
	graphRoot      string
	graphDriver    string
	graphOptions   []string
	loaded         bool
	layerStore     layer.Store
	imageStore     image.Store
	referenceStore reference.Store
	containerStore container.Store
	petStore       PetStore
	imageExporter  image.Exporter
}

func MakeMall(graphRoot, graphDriver string, graphOptions []string) Mall {
	return &mall{
		graphRoot:    graphRoot,
		graphDriver:  graphDriver,
		graphOptions: graphOptions,
	}
}

func (m *mall) GetGraphDriver() string {
	return m.graphDriver
}

func (m *mall) load() error {
	options := layer.StoreOptions{
		StorePath:                 m.graphRoot,
		MetadataStorePathTemplate: filepath.Join(m.graphRoot, "image", "%s", "layerdb"),
		GraphDriver:               m.graphDriver,
		GraphDriverOptions:        m.graphOptions,
	}
	ls, err := layer.NewStoreFromOptions(options)
	if err != nil {
		return err
	}
	m.layerStore = ls

	ipath := filepath.Join(m.graphRoot, "image", m.graphDriver, "imagedb")
	if err := os.MkdirAll(ipath, 0700); err != nil {
		return err
	}
	isb, err := image.NewFSStoreBackend(ipath)
	if err != nil {
		return err
	}

	is, err := image.NewImageStore(isb, ls)
	if err != nil {
		return err
	}
	m.imageStore = is

	rpath := filepath.Join(m.graphRoot, "image", m.graphDriver, "repositories.json")
	rs, err := reference.NewReferenceStore(rpath)
	if err != nil {
		return err
	}
	m.referenceStore = rs

	cpath := filepath.Join(m.graphRoot, "containers")
	if err := os.MkdirAll(cpath, 0700); err != nil {
		return err
	}
	ids, err := ioutil.ReadDir(cpath)
	if err != nil {
		return err
	}
	cs := container.NewMemoryStore()
	for _, file := range ids {
		path := filepath.Join(cpath, file.Name())
		c := container.NewBaseContainer(file.Name(), path)
		err = c.FromDisk()
		if err != nil {
			return err
		}
		cs.Add(c.ID, c)
	}
	m.containerStore = cs

	ppath := filepath.Join(m.graphRoot, "pets")
	if err := os.MkdirAll(ppath, 0700); err != nil {
		return err
	}
	ps, err := newPetStore(ppath, ls)
	if err != nil {
		return err
	}
	m.petStore = ps

	iexporter := tarexport.NewTarExporter(is, ls, rs, &imageEventLogger{log: logImageEvent})
	m.imageExporter = iexporter

	m.loaded = true
	return nil
}

func (m *mall) GetLayerStore() (layer.Store, error) {
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

func (m *mall) GetReferenceStore() (reference.Store, error) {
	if !m.loaded {
		if err := m.load(); err != nil {
			return nil, err
		}
	}
	if m.referenceStore != nil {
		return m.referenceStore, nil
	}
	return nil, LoadError
}

func (m *mall) GetImageStore() (image.Store, error) {
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

func (m *mall) GetContainerStore() (container.Store, error) {
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

type imageEventLogger struct {
	log func(imageID, refName, action string)
}

func (i *imageEventLogger) LogImageEvent(imageID, refName, action string) {
	if i.log != nil {
		i.log(imageID, refName, action)
	}
}

func (m *mall) GetImageExporter(logImageEvent func(imageID, refName, action string)) (image.Exporter, error) {
	if !m.loaded {
		if err := m.load(); err != nil {
			return nil, err
		}
	}
	if m.imageExporter != nil {
		return m.imageExporter, nil
	}
	return nil, LoadError
}

func (m *mall) GetPetStore() (PetStore, error) {
	if !m.loaded {
		if err := m.load(); err != nil {
			return nil, err
		}
	}
	if m.petStore != nil {
		return m.petStore, nil
	}
	return nil, LoadError
}
