//go:build linux && cgo
// +build linux,cgo

package loopback

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxDevicesPerGoroutine = 1000
	maxGoroutines          = 10
)

func TestAttachLoopbackDeviceRace(t *testing.T) {
	createLoopbackDevice := func() {
		// Create a file to use as a backing file
		f, err := os.CreateTemp(t.TempDir(), "loopback-test")
		require.NoError(t, err)
		defer f.Close()

		defer os.Remove(f.Name())

		lp, err := AttachLoopDevice(f.Name())
		assert.NoError(t, err)
		assert.NotNil(t, lp, "loopback device file should not be nil")
		if lp != nil {
			lp.Close()
		}
	}

	wg := sync.WaitGroup{}

	for i := 0; i < maxGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < maxDevicesPerGoroutine; i++ {
				createLoopbackDevice()
			}
		}()
	}
	wg.Wait()
}
