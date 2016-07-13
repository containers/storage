package cow

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/daemon/graphdriver"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/stringid"
)

var (
	ErrParentUnknown = errors.New("parent of layer not known")
	ErrLayerUnknown  = errors.New("layer not known")
)

// A Layer is a record of a copy-on-write layer that's stored by the lower
// level graph driver.
// ID is either one specified at import-time or a randomly-generated value.
// Name is an optional user-defined convenience value.  Parent is the ID of a
// layer from which this layer inherits data.  MountLabel is an SELinux label
// which should be used when attempting to mount the layer.  MountPoint is the
// path where the layer is mounted, or where it was most recently mounted.
type Layer struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Parent     string `json:"parent,omitempty"`
	MountLabel string `json:"mountlabel,omitempty"`
	MountPoint string `json:"mountpoint,omitempty"`
}

// LayerStore wraps a graph driver, adding the ability to refer to layers by
// name, and keeping track of parent-child relationships, along with a list of
// all known layers.
//
// Create creates a new layer, optionally giving it a specified ID rather than
// a randomly-generated one, either inheriting data from another specified
// layer or the empty base layer.  The new layer can optionally be given a name
// and have an SELinux label specified for use when mounting it.  Some
// underlying drivers can accept a "size" option.  At this time, drivers do not
// themselves distinguish between writeable and read-only layers.
//
// Exists checks if a layer with the specified name or ID is known.
//
// Status returns an slice of key-value pairs, suitable for human consumption,
// relaying whatever status information the driver can share.
//
// Delete deletes a layer with the specified name or ID.
//
// Wipe deletes all layers.
//
// Mount mounts a layer for use.  If the specified layer is the parent of other
// layers, it should not be written to.  An SELinux label to be applied to the
// mount can be specified to override the one configured for the layer.
//
// Unmount unmounts a layer when it is no longer in use.
//
// Changes returns a slice of Change structures, which contain a pathname
// (Path) and a description of what sort of change (Kind) was made by the
// layer (either ChangeModify, ChangeAdd, or ChangeDelete), relative to a
// specified layer.  By default, the layer's parent is used as a reference.
//
// Diff produces a tarstream which can be applied to a layer with the contents
// of the first layer to produce a layer with the contents of the second layer.
// By default, the parent of the second layer is used as the first layer.
//
// DiffSize produces an estimate of the length of the tarstream which would be
// produced by Diff.
//
// ApplyDiff reads tarstream which was created by a previous call to Diff and
// applies its changes to a specified layer.
//
// Lookup attempts to translate a name to an ID.  Most methods do this
// implicitly.
//
// Layers returns a slice of the known layers.
//
type LayerStore interface {
	Create(id, parent, name, mountLabel string, options map[string]string, writeable bool) (*Layer, error)
	Exists(id string) bool
	Get(id string) (*Layer, error)
	Status() ([][2]string, error)
	Delete(id string) error
	Wipe() error
	Mount(id, mountLabel string) (string, error)
	Unmount(id string) error
	Changes(from, to string) ([]archive.Change, error)
	Diff(from, to string) (archive.Reader, error)
	DiffSize(from, to string) (int64, error)
	ApplyDiff(to string, diff archive.Reader) (int64, error)
	Lookup(name string) (string, error)
	Layers() ([]Layer, error)
}

type layerStore struct {
	driver   graphdriver.Driver
	dir      string
	layers   []Layer
	byid     map[string]*Layer
	byname   map[string]*Layer
	byparent map[string][]*Layer
	bymount  map[string]*Layer
}

func (r *layerStore) Layers() ([]Layer, error) {
	return r.layers, nil
}

func (r *layerStore) Load() error {
	rpath := filepath.Join(r.dir, "layers.json")
	data, err := ioutil.ReadFile(rpath)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else {
		layers := []Layer{}
		ids := make(map[string]*Layer)
		names := make(map[string]*Layer)
		mounts := make(map[string]*Layer)
		parents := make(map[string][]*Layer)
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
					parents[layer.Parent] = []*Layer{&layers[n]}
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

func (r *layerStore) Save() error {
	rpath := filepath.Join(r.dir, "layers.json")
	jdata, err := json.Marshal(&r.layers)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(rpath, jdata, 0600)
}

func newLayerStore(dir string, driver graphdriver.Driver) (LayerStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	rlstore := layerStore{
		driver:   driver,
		dir:      dir,
		byid:     make(map[string]*Layer),
		bymount:  make(map[string]*Layer),
		byname:   make(map[string]*Layer),
		byparent: make(map[string][]*Layer),
	}
	if err := rlstore.Load(); err != nil {
		return nil, err
	}
	return &rlstore, nil
}

func (r *layerStore) Status() ([][2]string, error) {
	return r.driver.Status(), nil
}

func (r *layerStore) Create(id, parent, name, mountLabel string, options map[string]string, writeable bool) (layer *Layer, err error) {
	if layer, ok := r.byname[parent]; ok {
		parent = layer.ID
	}
	if id == "" {
		id = stringid.GenerateRandomID()
	}
	if _, nameInUse := r.byname[name]; nameInUse {
		return nil, DuplicateName
	}
	if writeable {
		err = r.driver.CreateReadWrite(id, parent, mountLabel, options)
	} else {
		err = r.driver.Create(id, parent, mountLabel, options)
	}
	if err == nil {
		newLayer := Layer{
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
			r.byparent[parent] = []*Layer{layer}
		}
		err = r.Save()
	}
	return layer, err
}

func (r *layerStore) Mount(id, mountLabel string) (string, error) {
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

func (r *layerStore) Unmount(id string) error {
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

func (r *layerStore) Delete(id string) error {
	if layer, ok := r.byname[id]; ok {
		id = layer.ID
	}
	r.Unmount(id)
	err := r.driver.Remove(id)
	if err == nil {
		if layer, ok := r.byid[id]; ok {
			pslice := r.byparent[layer.Parent]
			newPslice := []*Layer{}
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
			newLayers := []Layer{}
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

func (r *layerStore) Lookup(name string) (id string, err error) {
	layer, ok := r.byname[name]
	if !ok {
		return "", ErrLayerUnknown
	}
	return layer.ID, nil
}

func (r *layerStore) Exists(id string) bool {
	if layer, ok := r.byname[id]; ok {
		id = layer.ID
	}
	return r.driver.Exists(id)
}

func (r *layerStore) Get(id string) (*Layer, error) {
	if l, ok := r.byname[id]; ok {
		return l, nil
	}
	if l, ok := r.byid[id]; ok {
		return l, nil
	}
	return nil, ErrLayerUnknown
}

func (r *layerStore) Wipe() error {
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

func (r *layerStore) Changes(to, from string) ([]archive.Change, error) {
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

func (r *layerStore) Diff(to, from string) (archive.Reader, error) {
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

func (r *layerStore) DiffSize(to, from string) (size int64, err error) {
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

func (r *layerStore) ApplyDiff(to string, diff archive.Reader) (size int64, err error) {
	if layer, ok := r.byname[to]; ok {
		to = layer.ID
	}
	if layer, ok := r.byid[to]; !ok {
		return -1, ErrParentUnknown
	} else {
		return r.driver.ApplyDiff(layer.ID, layer.Parent, diff)
	}
}
