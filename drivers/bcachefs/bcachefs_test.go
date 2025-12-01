//go:build linux

package bcachefs

import (
	"os"
	"path/filepath"
	"testing"

	graphdriver "github.com/containers/storage/drivers"
)

func TestBcachefsSubvolumeOperations(t *testing.T) {
	testRoot := os.Getenv("BCACHEFS_TEST_ROOT")
	if testRoot == "" {
		t.Skip("BCACHEFS_TEST_ROOT not set, skipping bcachefs tests")
	}

	testDir := filepath.Join(testRoot, "test-subvol")

	t.Run("CreateSubvolume", func(t *testing.T) {
		err := subvolCreate(testDir)
		if err != nil {
			t.Fatalf("Failed to create subvolume: %v", err)
		}

		if _, err := os.Stat(testDir); os.IsNotExist(err) {
			t.Fatal("Subvolume directory does not exist after creation")
		}
	})

	t.Run("IsSubvolume", func(t *testing.T) {
		isSub, err := isSubvolume(testDir)
		if err != nil {
			t.Fatalf("Failed to check isSubvolume: %v", err)
		}
		if !isSub {
			t.Error("Expected testDir to be detected as a subvolume")
		}

		isSub, err = isSubvolume(testRoot)
		if err != nil {
			t.Fatalf("Failed to check isSubvolume for testRoot: %v", err)
		}
		if !isSub {
			t.Error("Expected testRoot to be detected as a subvolume")
		}
	})

	t.Run("CreateSnapshot", func(t *testing.T) {
		snapshotDir := filepath.Join(testRoot, "test-snapshot")
		err := subvolSnapshot(testDir, snapshotDir)
		if err != nil {
			t.Fatalf("Failed to create snapshot: %v", err)
		}

		if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
			t.Fatal("Snapshot directory does not exist after creation")
		}

		err = subvolDelete(testRoot, "test-snapshot")
		if err != nil {
			t.Fatalf("Failed to delete snapshot: %v", err)
		}
	})

	t.Run("DeleteSubvolume", func(t *testing.T) {
		err := subvolDelete(testRoot, "test-subvol")
		if err != nil {
			t.Fatalf("Failed to delete subvolume: %v", err)
		}

		if _, err := os.Stat(testDir); !os.IsNotExist(err) {
			t.Fatal("Subvolume directory still exists after deletion")
		}
	})
}

func TestBcachefsDriverInit(t *testing.T) {
	testRoot := os.Getenv("BCACHEFS_TEST_ROOT")
	if testRoot == "" {
		t.Skip("BCACHEFS_TEST_ROOT not set, skipping bcachefs tests")
	}

	if os.Getuid() != 0 {
		t.Skip("Driver init requires root (for mount.MakePrivate)")
	}

	driverHome := filepath.Join(testRoot, "driver-test")
	if err := os.MkdirAll(driverHome, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(driverHome)

	driver, err := Init(driverHome, graphdriver.Options{})
	if err != nil {
		t.Fatalf("Failed to initialize driver: %v", err)
	}

	if driver.String() != "bcachefs" {
		t.Errorf("Expected driver name 'bcachefs', got '%s'", driver.String())
	}
}
