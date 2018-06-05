// The dummy driver allows the use of storage for containers but without the need of managing images.
package dummy

import (
	"github.com/containers/storage/drivers"
	"github.com/containers/storage/pkg/idtools"
)

func init() {
	graphdriver.Register("dummy", Init)
}

// Init returns a new DUMMY driver.
// This sets the home directory for the driver and returns NaiveDiffDriver.
func Init(home string, options []string, uidMaps, gidMaps []idtools.IDMap) (graphdriver.Driver, error) {
	d := &Driver{}
	return graphdriver.NewNaiveDiffDriver(d, graphdriver.NewNaiveLayerIDMapUpdater(d)), nil
}

type Driver struct {
}

func (d *Driver) String() string {
	return "dummy"
}

func (d *Driver) Status() [][2]string {
	return nil
}

// Metadata is used for implementing the graphdriver.ProtoDriver interface. DUMMY does not currently have any meta data.
func (d *Driver) Metadata(id string) (map[string]string, error) {
	return nil, nil
}

// Cleanup is used to implement graphdriver.ProtoDriver. There is no cleanup required for this driver.
func (d *Driver) Cleanup() error {
	return nil
}

// CreateReadWrite creates a layer that is writable for use as a container
// file system.
func (d *Driver) CreateReadWrite(id, parent string, opts *graphdriver.CreateOpts) error {
	return nil
}

// Create prepares the filesystem for the DUMMY driver and copies the directory for the given id under the parent.
func (d *Driver) Create(id, parent string, opts *graphdriver.CreateOpts) error {
	return nil
}

// Remove deletes the content from the directory for a given id.
func (d *Driver) Remove(id string) error {
	return nil
}

// Get returns the directory for the given id.
func (d *Driver) Get(id, mountLabel string) (string, error) {
	return id, nil
}

// Put is a noop for dummy that return nil for the error, since this driver has no runtime resources to clean up.
func (d *Driver) Put(id string) error {
	return nil
}

// Exists checks to see if the directory exists for the given id.
func (d *Driver) Exists(id string) bool {
	return true
}

// AdditionalImageStores returns additional image stores supported by the driver
func (d *Driver) AdditionalImageStores() []string {
	return nil
}
