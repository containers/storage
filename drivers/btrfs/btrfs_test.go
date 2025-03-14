//go:build linux && cgo

package btrfs

import (
	"os"
	"path"
	"testing"

	graphdriver "github.com/containers/storage/drivers"
	"github.com/containers/storage/drivers/graphtest"
)

// This avoids creating a new driver for each test if all tests are run
// Make sure to put new tests between TestBtrfsSetup and TestBtrfsTeardown
func TestBtrfsSetup(t *testing.T) {
	graphtest.GetDriverNoCleanup(t, "btrfs")
}

func TestBtrfsCreateEmpty(t *testing.T) {
	graphtest.DriverTestCreateEmpty(t, "btrfs")
}

func TestBtrfsCreateBase(t *testing.T) {
	graphtest.DriverTestCreateBase(t, "btrfs")
}

func TestBtrfsCreateSnap(t *testing.T) {
	graphtest.DriverTestCreateSnap(t, "btrfs")
}

func TestBtrfsCreateFromTemplate(t *testing.T) {
	graphtest.DriverTestCreateFromTemplate(t, "btrfs")
}

func TestBtrfsSubvolDelete(t *testing.T) {
	d := graphtest.GetDriver(t, "btrfs")
	if err := d.CreateReadWrite("test", "", nil); err != nil {
		t.Fatal(err)
	}

	dir, err := d.Get("test", graphdriver.MountOpts{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := d.Put("test"); err != nil {
			t.Fatal(err)
		}
	}()

	if err := subvolCreate(dir, "subvoltest"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path.Join(dir, "subvoltest")); err != nil {
		t.Fatal(err)
	}

	if err := d.Remove("test"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path.Join(dir, "subvoltest")); !os.IsNotExist(err) {
		t.Fatalf("expected not exist error on nested subvol, got: %v", err)
	}
}

func TestBtrfsEcho(t *testing.T) {
	graphtest.DriverTestEcho(t, "btrfs")
}

func TestBtrfsListLayers(t *testing.T) {
	graphtest.DriverTestListLayers(t, "btrfs")
}

func TestBtrfsTeardown(t *testing.T) {
	graphtest.PutDriver(t)
}
