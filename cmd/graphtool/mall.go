package main

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/digest"
	"github.com/docker/docker/container"
	_ "github.com/docker/docker/daemon/graphdriver/register"
	"github.com/docker/docker/image"
	"github.com/docker/docker/image/tarexport"
	"github.com/docker/docker/layer"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/reference"
)

var (
	LoadError        = errors.New("error loading storage metadata")
	DuplicatePetName = errors.New("name for pet layer already in use")
)

type Mall interface {
	GetGraphDriver() string
	GetLayerStore() (layer.Store, error)
	GetImageStore() (image.Store, error)
	GetReferenceStore() (reference.Store, error)
	GetContainerStore() (container.Store, error)
	GetPetStore() (PetStore, error)
	GetImageExporter(func(string, string, string)) (image.Exporter, error)

	Images() (map[image.ID]*image.Image, map[*image.Image][]reference.Named, error)
	Containers() ([]*container.Container, error)
	Pets() ([]Pet, error)
	LoadImage(status io.Writer, quiet bool, images ...io.ReadCloser) error
	SaveImage(stream io.Writer, refs []string) error
	DeleteImage(refs []string) error
	CreatePet(imageRef, petName, mountLabel string) (petID string, err error)
	DeletePet(nameOrID string) error
	Mount(nameOrID string) (path string, err error)
	Unmount(nameOrID string) error
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

func (m *mall) Containers() ([]*container.Container, error) {
	cstore, err := m.GetContainerStore()
	if err != nil {
		return nil, err
	}
	return cstore.List(), nil
}

func (m *mall) Images() (map[image.ID]*image.Image, map[*image.Image][]reference.Named, error) {
	istore, err := m.GetImageStore()
	if err != nil {
		return nil, nil, err
	}
	rstore, err := m.GetReferenceStore()
	if err != nil {
		return nil, nil, err
	}
	images := istore.Map()
	refs := make(map[*image.Image][]reference.Named)
	for id, image := range images {
		refs[image] = rstore.References(id)
	}
	return images, refs, nil
}

func (m *mall) Pets() ([]Pet, error) {
	pstore, err := m.GetPetStore()
	if err != nil {
		return nil, err
	}
	return pstore.List()
}

func (m *mall) LoadImage(status io.Writer, quiet bool, images ...io.ReadCloser) error {
	e, err := m.GetImageExporter(logImageEvent)
	if err != nil {
		return err
	}
	for _, image := range images {
		if err == nil {
			err = e.Load(image, status, quiet)
		}
		image.Close()
	}
	return err
}

func (m *mall) SaveImage(stream io.Writer, refs []string) error {
	e, err := m.GetImageExporter(logImageEvent)
	if err != nil {
		return err
	}
	return e.Save(refs, stream)
}

func (m *mall) resolveImage(imageRef string) (imageID image.ID, layerID layer.ChainID, imageName string, err error) {
	var img *image.Image
	istore, err := m.GetImageStore()
	if err != nil {
		return "", "", "", err
	}
	imageid, ref, err := reference.ParseIDOrReference(imageRef)
	if err != nil {
		logrus.Debugf("Error parsing ID or reference %q: %v", imageRef, err)
		roid, err := digest.ParseDigest(imageRef)
		if err != nil {
			logrus.Debugf("Error parsing %q as digest: %v", imageRef, err)
			return "", "", "", err
		}
		layerID = layer.ChainID(roid)
		imageID = image.ID(roid)
		logrus.Debugf("Resolved %q to ID %q.", imageRef, imageID)
		img, err = istore.Get(imageID)
		if err != nil {
			return "", "", "", err
		}
	} else {
		rstore, err := m.GetReferenceStore()
		if err != nil {
			return "", "", "", err
		}
		roid, err := digest.ParseDigest(imageid.String())
		if err != nil {
			associations := rstore.ReferencesByName(ref)
			if len(associations) == 0 {
				logrus.Debugf("No image with name %q.", ref.String())
			} else {
				logrus.Debugf("Resolved %q to name %q.", imageRef, ref)
				for _, association := range associations {
					logrus.Debugf("Attempting to use ID %q.", association.ImageID)
					img, err = istore.Get(association.ImageID)
					if err == nil {
						imageName = association.Ref.FullName()
						break
					}
					logrus.Debugf("No image with ID %s", association.ImageID)
				}
			}
		} else {
			logrus.Debugf("Resolved %q to ID %q.", imageRef, roid)
			img, err = istore.Get(image.ID(roid))
		}
		if err != nil {
			return "", "", "", err
		}
	}
	if img == nil {
		logrus.Debugf("No image matched %s", imageRef)
		return "", "", "", reference.ErrDoesNotExist
	}
	rootfs := img.RootFS
	if rootfs.Type != "layers" {
		logrus.Debugf("Don't know how to deal with rootfs type %q, only layers.", rootfs.Type)
		return "", "", "", layer.ErrNotSupported
	}
	if len(rootfs.DiffIDs) == 0 {
		logrus.Debugf("No layers in this image, trying anyway.")
	}
	layerID = rootfs.ChainID()
	imageID = img.ID()
	return imageID, layerID, imageName, nil
}

func (m *mall) DeleteImage(refs []string) error {
	istore, err := m.GetImageStore()
	if err != nil {
		return err
	}
	rstore, err := m.GetReferenceStore()
	if err != nil {
		return err
	}
	for _, ref := range refs {
		imageID, _, _, err := m.resolveImage(ref)
		if err != nil {
			return err
		}
		metadata, err := istore.Delete(imageID)
		if err != nil {
			return err
		}
		layer.LogReleaseMetadata(metadata)
		for _, ref := range rstore.References(imageID) {
			deleted, err := rstore.Delete(ref)
			if err != nil {
				return err
			}
			if !deleted {
				logrus.Errorf("Unable to remove reference %q to %s.", ref, imageID.String())
			} else {
				logrus.Debugf("Removed reference %q to %s.", ref, imageID.String())
			}
		}
	}
	return nil
}

func (m *mall) CreatePet(imageRef, petName, mountLabel string) (petID string, err error) {
	lstore, err := m.GetLayerStore()
	if err != nil {
		return "", err
	}
	pstore, err := m.GetPetStore()
	if err != nil {
		return "", err
	}
	if petName != "" {
		if _, err := pstore.Get(petName); err == nil {
			return "", DuplicatePetName
		}
	}
	imageID, layerID, imageName, err := m.resolveImage(imageRef)
	if err != nil {
		return "", err
	}
	rwlayerID := stringid.GenerateRandomID()
	options := make(map[string]string)
	rwlayer, err := lstore.CreateRWLayer(rwlayerID, layerID, mountLabel, nil, options)
	if err != nil {
		logrus.Debugf("Error creating new layer from %q: %v.", layerID, err)
		return "", err
	}
	petID = stringid.GenerateRandomID()
	pet, err := pstore.Add(petID, petName, imageID.String(), imageName, rwlayer, mountLabel)
	if err != nil {
		logrus.Debugf("Error registering new pet layer: %v.", err)
		return "", err
	}
	return pet.ID(), nil
}

func (m *mall) DeletePet(nameOrID string) error {
	pstore, err := m.GetPetStore()
	if err != nil {
		return err
	}
	p, err := pstore.Get(nameOrID)
	if err != nil {
		return err
	}
	err = pstore.Remove(p.ID(), p.Name())
	if err != nil {
		return err
	}
	return nil
}

func (m *mall) getRWLayer(nameOrID string) (layer layer.RWLayer, mountLabel string, err error) {
	pstore, err := m.GetPetStore()
	if err != nil {
		return nil, "", err
	}
	cstore, err := m.GetContainerStore()
	if err != nil {
		return nil, "", err
	}
	layerStore, err := m.GetLayerStore()
	if err != nil {
		return nil, "", err
	}
	pets, err := pstore.List()
	if err != nil {
		return nil, "", err
	}
	for _, pet := range pets {
		if petMatch(pet, nameOrID) {
			return pet.Layer(), pet.MountLabel(), nil
		}
	}
	for _, container := range cstore.List() {
		if containerMatch(container, nameOrID) {
			layer, err = layerStore.GetRWLayer(container.ID)
			if err != nil {
				return nil, "", err
			}
			return layer, container.GetMountLabel(), nil
		}
	}
	layer, err = layerStore.GetRWLayer(nameOrID)
	if err != nil {
		return nil, "", err
	}
	if layer == nil {
		return nil, "", noMatchingContainerError
	}
	return layer, "", nil
}

func (m *mall) Mount(nameOrID string) (path string, err error) {
	layer, label, err := m.getRWLayer(nameOrID)
	if err != nil {
		return "", err
	}
	return layer.Mount(label)
}

func (m *mall) Unmount(nameOrID string) error {
	layer, _, err := m.getRWLayer(nameOrID)
	if err != nil {
		return err
	}
	return layer.Unmount()
}
