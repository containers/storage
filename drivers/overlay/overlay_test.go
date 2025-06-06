//go:build linux

package overlay

import (
	"os"
	"testing"

	graphdriver "github.com/containers/storage/drivers"
	"github.com/containers/storage/drivers/graphtest"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/reexec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const driverName = "overlay"

func init() {
	// Do not sure chroot to speed run time and allow archive
	// errors or hangs to be debugged directly from the test process.
	untar = archive.UntarUncompressed
	graphdriver.ApplyUncompressedLayer = archive.ApplyUncompressedLayer

	reexec.Init()
}

func skipIfNaive(t *testing.T) {
	td := t.TempDir()

	if err := doesSupportNativeDiff(td, ""); err != nil {
		t.Skipf("Cannot run test with naive diff")
	}
}

// Ensure that a layer created with force_mask will keep the root directory mode
// with user.containers.override_stat. This preserved mode should also be
// inherited by the upper layer, whether force_mask is set or not.
//
// This test is placed before TestOverlaySetup() because it uses driver options
// different from the other tests.
func TestContainersOverlayXattr(t *testing.T) {
	driver := graphtest.GetDriver(t, driverName, "force_mask=700")
	require.NoError(t, driver.Create("lower", "", nil))
	graphtest.ReconfigureDriver(t, driverName)
	require.NoError(t, driver.Create("upper", "lower", nil))

	root, err := driver.Get("upper", graphdriver.MountOpts{})
	require.NoError(t, err)
	fi, err := os.Stat(root)
	require.NoError(t, err)
	assert.Equal(t, 0o555&os.ModePerm, fi.Mode()&os.ModePerm, root)
}

func TestSupportsShifting(t *testing.T) {
	contiguousMap := []idtools.IDMap{
		{
			ContainerID: 0,
			HostID:      1000,
			Size:        65536,
		},
	}
	nonContiguousMap := []idtools.IDMap{
		{
			ContainerID: 0,
			HostID:      0,
			Size:        1,
		},
		{
			ContainerID: 2,
			HostID:      2,
			Size:        1,
		},
	}

	t.Run("no mount program", func(t *testing.T) {
		driver := graphtest.GetDriver(t, driverName)

		supported := driver.SupportsShifting(nil, nil)
		assert.Equal(t, supported, driver.SupportsShifting(contiguousMap, contiguousMap), "contiguous map with no mount program")
		assert.Equal(t, supported, driver.SupportsShifting(nonContiguousMap, nonContiguousMap), "non-contiguous map with no mount program")
	})

	t.Run("with mount program", func(t *testing.T) {
		driver := graphtest.GetDriver(t, driverName, "mount_program=/usr/bin/true")

		assert.True(t, driver.SupportsShifting(nil, nil), "nil map with mount program")
		assert.True(t, driver.SupportsShifting(contiguousMap, contiguousMap), "contiguous map with mount program")
		// If a mount program is specified, SupportsShifting must return false
		assert.False(t, driver.SupportsShifting(nonContiguousMap, nonContiguousMap), "non-contiguous map with mount program")
	})
}

// This avoids creating a new driver for each test if all tests are run
// Make sure to put new tests between TestOverlaySetup and TestOverlayTeardown
func TestOverlaySetup(t *testing.T) {
	graphtest.GetDriver(t, driverName)
}

func TestOverlayCreateEmpty(t *testing.T) {
	graphtest.DriverTestCreateEmpty(t, driverName)
}

func TestOverlayCreateBase(t *testing.T) {
	graphtest.DriverTestCreateBase(t, driverName)
}

func TestOverlayCreateSnap(t *testing.T) {
	graphtest.DriverTestCreateSnap(t, driverName)
}

func TestOverlayCreateFromTemplate(t *testing.T) {
	graphtest.DriverTestCreateFromTemplate(t, driverName)
}

func TestOverlay128LayerRead(t *testing.T) {
	graphtest.DriverTestDeepLayerRead(t, 128, driverName)
}

func TestOverlayDiffApply10Files(t *testing.T) {
	skipIfNaive(t)
	graphtest.DriverTestDiffApply(t, 10, driverName)
}

func TestOverlayChanges(t *testing.T) {
	skipIfNaive(t)
	graphtest.DriverTestChanges(t, driverName)
}

func TestOverlayEcho(t *testing.T) {
	graphtest.DriverTestEcho(t, driverName)
}

func TestOverlayListLayers(t *testing.T) {
	graphtest.DriverTestListLayers(t, driverName)
}

func TestOverlayTeardown(t *testing.T) {
	graphtest.PutDriver(t)
}

// Benchmarks should always setup new driver

func BenchmarkExists(b *testing.B) {
	graphtest.DriverBenchExists(b, driverName)
}

func BenchmarkGetEmpty(b *testing.B) {
	graphtest.DriverBenchGetEmpty(b, driverName)
}

func BenchmarkDiffBase(b *testing.B) {
	graphtest.DriverBenchDiffBase(b, driverName)
}

func BenchmarkDiffSmallUpper(b *testing.B) {
	graphtest.DriverBenchDiffN(b, 10, 10, driverName)
}

func BenchmarkDiff10KFileUpper(b *testing.B) {
	graphtest.DriverBenchDiffN(b, 10, 10000, driverName)
}

func BenchmarkDiff10KFilesBottom(b *testing.B) {
	graphtest.DriverBenchDiffN(b, 10000, 10, driverName)
}

func BenchmarkDiffApply100(b *testing.B) {
	graphtest.DriverBenchDiffApplyN(b, 100, driverName)
}

func BenchmarkDiff20Layers(b *testing.B) {
	graphtest.DriverBenchDeepLayerDiff(b, 20, driverName)
}

func BenchmarkRead20Layers(b *testing.B) {
	graphtest.DriverBenchDeepLayerRead(b, 20, driverName)
}
