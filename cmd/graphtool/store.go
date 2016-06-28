package main

import (
	"encoding/json"
	"errors"
	"github.com/docker/docker/pkg/ioutils"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/layer"
)

var (
	NoSuchPetError   = errors.New("no such pet found")
	NoLayerNameError = errors.New("pet has no layer")
)

type pet struct {
	petFile       string
	PetID         string `json:"id"`
	PetName       string `json:"name,omitempty"`
	PetMountLabel string `json:"mountlabel,omitempty"`
	PetImageID    string `json:"imageid"`
	PetImageName  string `json:"imagename,omitempty"`
	PetLayerName  string `json:"layer"`
	petLayer      layer.RWLayer
}

type Pet interface {
	ID() string
	ImageID() string
	Name() string
	MountLabel() string
	Layer() layer.RWLayer
}

type petStore struct {
	dir    string
	ls     layer.Store
	byid   map[string]*pet
	byname map[string]*pet
}

type PetStore interface {
	Add(ID, name, imageID, imageName string, layer layer.RWLayer, mountLabel string) (Pet, error)
	Remove(ID, name string) error
	Get(IDorName string) (Pet, error)
	Load() error
	Save() error
	List() ([]Pet, error)
}

func (p *petStore) loadPet(path string) (*pet, error) {
	tmp := &pet{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(tmp)
	if err != nil {
		return nil, err
	}
	if tmp.PetLayerName == "" {
		return nil, NoLayerNameError
	}
	tmp.petLayer, err = p.ls.GetRWLayer(tmp.PetLayerName)
	if err != nil {
		return nil, err
	}
	tmp.petFile = path
	if err != nil {
		return nil, err
	}
	return tmp, nil
}

func (p *petStore) savePet(pet *pet) error {
	f, err := ioutils.NewAtomicFileWriter(pet.petFile, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(*pet)
}

func (p *pet) ID() string {
	return p.PetID
}

func (p *pet) Name() string {
	return p.PetName
}

func (p *pet) ImageID() string {
	return p.PetImageID
}

func (p *pet) ImageName() string {
	return p.PetImageName
}

func (p *pet) MountLabel() string {
	return p.PetMountLabel
}

func (p *pet) Layer() layer.RWLayer {
	return p.petLayer
}

func (p *petStore) Load() error {
	ids, err := ioutil.ReadDir(p.dir)
	if err != nil {
		return err
	}
	unseen := map[string]string{}
	for id, _ := range p.byid {
		unseen[id] = id
	}
	for _, file := range ids {
		path := filepath.Join(p.dir, file.Name())
		pet, err := p.loadPet(path)
		if err != nil {
			return err
		}
		p.byid[pet.PetID] = pet
		if pet.PetName != "" {
			p.byname[pet.PetName] = pet
		}
		delete(unseen, pet.PetID)
	}
	for id, _ := range unseen {
		p.Remove(id, "")
	}
	return nil
}

func (p *petStore) Save() error {
	for _, pet := range p.byid {
		err := p.savePet(pet)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *petStore) Add(ID, name, imageID, imageName string, layer layer.RWLayer, mountLabel string) (Pet, error) {
	path := filepath.Join(p.dir, ID)
	pet := pet{
		petFile:       path,
		PetID:         ID,
		PetName:       name,
		PetImageID:    imageID,
		PetImageName:  imageName,
		PetLayerName:  layer.Name(),
		petLayer:      layer,
		PetMountLabel: mountLabel,
	}
	err := p.savePet(&pet)
	if err != nil {
		return nil, err
	}
	p.byid[pet.PetID] = &pet
	if pet.PetName != "" {
		p.byname[pet.PetName] = &pet
	}
	return &pet, err
}

func (p *petStore) Remove(ID, name string) error {
	var pet *pet
	if ID != "" {
		pet = p.byid[ID]
	}
	if pet == nil && name != "" {
		pet = p.byname[name]
	}
	if pet == nil {
		return NoSuchPetError
	}
	metadata, err := p.ls.ReleaseRWLayer(pet.petLayer)
	if err != nil {
		return err
	}
	layer.LogReleaseMetadata(metadata)
	err = os.Remove(pet.petFile)
	if err != nil {
		return err
	}
	delete(p.byid, pet.PetID)
	if pet.Name() != "" {
		delete(p.byname, pet.PetName)
	}
	return nil
}

func (p *petStore) Get(IDorName string) (Pet, error) {
	pet, ok := p.byid[IDorName]
	if !ok {
		pet = p.byname[IDorName]
	}
	if pet != nil {
		return pet, nil
	}
	return nil, NoSuchPetError
}

func (p *petStore) List() ([]Pet, error) {
	pets := []Pet{}
	for _, pet := range p.byid {
		pets = append(pets, pet)
	}
	return pets, nil
}

func newPetStore(dir string, ls layer.Store) (PetStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	p := petStore{dir: dir, ls: ls, byid: make(map[string]*pet), byname: make(map[string]*pet)}
	if err := p.Load(); err != nil {
		return nil, err
	}
	return &p, nil
}
