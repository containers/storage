package cow

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/docker/docker/pkg/stringid"
)

var ErrContainerUnknown = errors.New("container not known")

// Container is a read-write layer with a configuration.
// ID is either one specified at create-time or a randomly-generated value.
// Name is an optional user-defined convenience value.
// ImageID is the ID of the image which was used to create the container.
// LayerID is the ID of the read-write layer for the container itself.
// It is expected that the image's last layer is the parent of the container's
// read-write layer.
type Container struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	ImageID  string `json:"image"`
	LayerID  string `json:"layer"`
	Metadata string `json:"metadata,omitempty"`
}

// ContainerStore provides bookkeeping for information about Containers.
//
// Create creates a container that has a specified ID (or a random one) and an
// optional name, based on the specified image, using the specified layer as
// its read-write layer.
//
// Get retrieves information about a container given an ID or name.
//
// Exists checks if there is a container with the given ID or name.
//
// Delete removes the record of the container.
//
// Wipe removes records of all containers.
//
// Containers returns a slice enumerating the known containers.
type ContainerStore interface {
	Create(id, name, image, layer, metadata string) (*Container, error)
	Get(id string) (*Container, error)
	Exists(id string) bool
	Delete(id string) error
	Wipe() error
	Containers() ([]Container, error)
}

type containerStore struct {
	dir        string
	containers []Container
	byid       map[string]*Container
	byname     map[string]*Container
}

func (r *containerStore) Containers() ([]Container, error) {
	return r.containers, nil
}

func (r *containerStore) Load() error {
	rpath := filepath.Join(r.dir, "containers.json")
	data, err := ioutil.ReadFile(rpath)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else {
		containers := []Container{}
		ids := make(map[string]*Container)
		names := make(map[string]*Container)
		if err = json.Unmarshal(data, &containers); len(data) == 0 || err == nil {
			for n, container := range containers {
				ids[container.ID] = &containers[n]
				if container.Name != "" {
					names[container.Name] = &containers[n]
				}
			}
		}
		r.containers = containers
		r.byid = ids
		r.byname = names
	}
	return nil
}

func (r *containerStore) Save() error {
	rpath := filepath.Join(r.dir, "containers.json")
	jdata, err := json.Marshal(&r.containers)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(rpath, jdata, 0600)
}

func newContainerStore(dir string) (ContainerStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	cstore := containerStore{
		dir:        dir,
		containers: []Container{},
		byid:       make(map[string]*Container),
		byname:     make(map[string]*Container),
	}
	if err := cstore.Load(); err != nil {
		return nil, err
	}
	return &cstore, nil
}

func (r *containerStore) Create(id, name, image, layer, metadata string) (container *Container, err error) {
	if id == "" {
		id = stringid.GenerateRandomID()
	}
	if err == nil {
		newContainer := Container{
			ID:       id,
			Name:     name,
			ImageID:  image,
			LayerID:  layer,
			Metadata: metadata,
		}
		r.containers = append(r.containers, newContainer)
		container = &r.containers[len(r.containers)-1]
		r.byid[id] = container
		if name != "" {
			r.byname[name] = container
		}
		err = r.Save()
	}
	return container, err
}

func (r *containerStore) Delete(id string) error {
	if container, ok := r.byname[id]; ok {
		id = container.ID
	}
	if _, ok := r.byid[id]; ok {
		newContainers := []Container{}
		for _, candidate := range r.containers {
			if candidate.ID != id {
				newContainers = append(newContainers, candidate)
			}
		}
		r.containers = newContainers
		if err := r.Save(); err != nil {
			return err
		}
	}
	return nil
}

func (r *containerStore) Get(id string) (*Container, error) {
	if c, ok := r.byname[id]; ok {
		return c, nil
	}
	if c, ok := r.byid[id]; ok {
		return c, nil
	}
	return nil, ErrContainerUnknown
}

func (r *containerStore) Exists(id string) bool {
	if _, ok := r.byname[id]; ok {
		return true
	}
	if _, ok := r.byid[id]; ok {
		return true
	}
	return false
}

func (r *containerStore) Wipe() error {
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
