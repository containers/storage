package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/daemon/graphdriver"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/ioutils"
)

var (
	ErrParentUnknown = errors.New("parent of layer not known")
	ErrLayerUnknown  = errors.New("layer not known")
)

type RawLayer struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Parent     string `json:"parent,omitempty"`
	MountLabel string `json:"mountlabel,omitempty"`
	MountPoint string `json:"mountpoint,omitempty"`
}

type rawLayerStore struct {
	driver   graphdriver.Driver
	dir      string
	layers   []RawLayer
	byid     map[string]*RawLayer
	byname   map[string]*RawLayer
	byparent map[string][]*RawLayer
	bymount  map[string]*RawLayer
}

type RawLayerStore interface {
	Create(id, parent, name, lastMountPoint string, options map[string]string, writeable bool) (*RawLayer, error)
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
	Lookup(name string) (string, error)
	Layers() ([]RawLayer, error)
}

func (r *rawLayerStore) Layers() ([]RawLayer, error) {
	return r.layers, nil
}

func (r *rawLayerStore) Load() error {
	rpath := filepath.Join(r.dir, "layers.json")
	data, err := ioutil.ReadFile(rpath)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else {
		layers := []RawLayer{}
		ids := make(map[string]*RawLayer)
		names := make(map[string]*RawLayer)
		mounts := make(map[string]*RawLayer)
		parents := make(map[string][]*RawLayer)
		if err = json.Unmarshal(data, &layers); len(data) == 0 || err == nil {
			for n, layer := range layers {
				ids[layer.ID] = &layers[n]
				if layer.Name != "" {
					names[layer.Name] = &layers[n]
				}
				if layer.MountPoint != "" {
					mounts[layer.MountPoint] = &layers[n]
				}
				if pslice, ok := parents[layer.Parent]; ok {
					parents[layer.Parent] = append(pslice, &layers[n])
				} else {
					parents[layer.Parent] = []*RawLayer{&layers[n]}
				}
			}
		}
		r.layers = layers
		r.byid = ids
		r.byname = names
		r.byparent = parents
		r.bymount = mounts
	}
	return nil
}

func (r *rawLayerStore) Save() error {
	rpath := filepath.Join(r.dir, "layers.json")
	jdata, err := json.Marshal(&r.layers)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(rpath, jdata, 0600)
}

func newRawLayerStore(dir string, driver graphdriver.Driver) (RawLayerStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	rlstore := rawLayerStore{
		driver:   driver,
		dir:      dir,
		byid:     make(map[string]*RawLayer),
		bymount:  make(map[string]*RawLayer),
		byname:   make(map[string]*RawLayer),
		byparent: make(map[string][]*RawLayer),
	}
	if err := rlstore.Load(); err != nil {
		return nil, err
	}
	return &rlstore, nil
}

func (r *rawLayerStore) Status() ([][2]string, error) {
	return r.driver.Status(), nil
}

func (r *rawLayerStore) Create(id, parent, name, mountLabel string, options map[string]string, writeable bool) (layer *RawLayer, err error) {
	if layer, ok := r.byname[parent]; ok {
		parent = layer.ID
	}
	if writeable {
		err = r.driver.CreateReadWrite(id, parent, mountLabel, options)
	} else {
		err = r.driver.Create(id, parent, mountLabel, options)
	}
	if err == nil {
		newLayer := RawLayer{
			ID:         id,
			Parent:     parent,
			Name:       name,
			MountLabel: mountLabel,
		}
		r.layers = append(r.layers, newLayer)
		layer = &r.layers[len(r.layers)-1]
		r.byid[id] = layer
		if name != "" {
			r.byname[name] = layer
		}
		if pslice, ok := r.byparent[parent]; ok {
			pslice = append(pslice, layer)
			r.byparent[parent] = pslice
		} else {
			r.byparent[parent] = []*RawLayer{layer}
		}
		err = r.Save()
	}
	return layer, err
}

func (r *rawLayerStore) Mount(id, mountLabel string) (string, error) {
	if layer, ok := r.byname[id]; ok {
		id = layer.ID
	}
	if mountLabel == "" {
		if layer, ok := r.byid[id]; ok {
			mountLabel = layer.MountLabel
		}
	}
	mountpoint, err := r.driver.Get(id, mountLabel)
	if mountpoint != "" && err == nil {
		if layer, ok := r.byid[id]; ok {
			if layer.MountPoint != "" {
				delete(r.bymount, layer.MountPoint)
			}
			layer.MountPoint = mountpoint
			r.bymount[layer.MountPoint] = layer
			err = r.Save()
		}
	}
	return mountpoint, err
}

func (r *rawLayerStore) Unmount(id string) error {
	if layer, ok := r.bymount[id]; ok {
		id = layer.ID
	}
	if layer, ok := r.byname[id]; ok {
		id = layer.ID
	}
	err := r.driver.Put(id)
	if err == nil {
		if layer, ok := r.byid[id]; ok {
			if layer.MountPoint != "" {
				delete(r.bymount, layer.MountPoint)
			}
			layer.MountPoint = ""
			err = r.Save()
		}
	}
	return err
}

func (r *rawLayerStore) Delete(id string) error {
	if layer, ok := r.byname[id]; ok {
		id = layer.ID
	}
	r.Unmount(id)
	err := r.driver.Remove(id)
	if err == nil {
		if layer, ok := r.byid[id]; ok {
			pslice := r.byparent[layer.Parent]
			newPslice := []*RawLayer{}
			for _, candidate := range pslice {
				if candidate.ID != id {
					newPslice = append(newPslice, candidate)
				}
			}
			if len(newPslice) > 0 {
				r.byparent[layer.Parent] = newPslice
			} else {
				delete(r.byparent, layer.Parent)
			}
			if layer.Name != "" {
				delete(r.byname, layer.Name)
			}
			if layer.MountPoint != "" {
				delete(r.bymount, layer.MountPoint)
			}
			newLayers := []RawLayer{}
			for _, candidate := range r.layers {
				if candidate.ID != id {
					newLayers = append(newLayers, candidate)
				}
			}
			r.layers = newLayers
			if err = r.Save(); err != nil {
				return err
			}
		}
	}
	return err
}

func (r *rawLayerStore) Lookup(name string) (id string, err error) {
	layer, ok := r.byname[name]
	if !ok {
		return "", ErrLayerUnknown
	}
	return layer.ID, nil
}

func (r *rawLayerStore) Exists(id string) bool {
	if layer, ok := r.byname[id]; ok {
		id = layer.ID
	}
	return r.driver.Exists(id)
}

func (r *rawLayerStore) Wipe() error {
	ids := []string{}
	for id, _ := range r.byid {
		ids = append(ids, id)
	}
	for _, id := range ids {
		if err := r.Delete(id); err != nil {
			return err
		}
	}
	return nil
}

func (r *rawLayerStore) Changes(to, from string) ([]archive.Change, error) {
	if layer, ok := r.byname[from]; ok {
		from = layer.ID
	}
	if layer, ok := r.byname[to]; ok {
		to = layer.ID
	}
	if from == "" {
		if layer, ok := r.byid[to]; ok {
			from = layer.Parent
		}
	}
	if to == "" {
		return nil, ErrParentUnknown
	}
	return r.driver.Changes(to, from)
}

func (r *rawLayerStore) Diff(to, from string) (archive.Reader, error) {
	if layer, ok := r.byname[from]; ok {
		from = layer.ID
	}
	if layer, ok := r.byname[to]; ok {
		to = layer.ID
	}
	if from == "" {
		if layer, ok := r.byid[to]; ok {
			from = layer.Parent
		}
	}
	if to == "" {
		return nil, ErrParentUnknown
	}
	return r.driver.Diff(to, from)
}

func (r *rawLayerStore) DiffSize(to, from string) (size int64, err error) {
	if layer, ok := r.byname[from]; ok {
		from = layer.ID
	}
	if layer, ok := r.byname[to]; ok {
		to = layer.ID
	}
	if from == "" {
		if layer, ok := r.byid[to]; ok {
			from = layer.Parent
		}
	}
	if to == "" {
		return -1, ErrParentUnknown
	}
	return r.driver.DiffSize(to, from)
}

func (r *rawLayerStore) ApplyDiff(to string, diff archive.Reader) (size int64, err error) {
	if layer, ok := r.byname[to]; ok {
		to = layer.ID
	}
	if layer, ok := r.byid[to]; !ok {
		return -1, ErrParentUnknown
	} else {
		return r.driver.ApplyDiff(layer.ID, layer.Parent, diff)
	}
}
