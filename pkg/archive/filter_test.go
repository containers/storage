package archive

import (
	"bufio"
	"bytes"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryProcFilter(t *testing.T) {
	t.Run("Invalid filter path", func(t *testing.T) {
		args := []string{"does-not-exist"}
		input := bufio.NewReader(bytes.NewBufferString("foo"))
		result, ok := tryProcFilter(args, input, func() {})
		assert.Nil(t, result)
		assert.False(t, ok)
	})

	t.Run("Valid filter path", func(t *testing.T) {
		inputData := "input data"

		args := []string{"cat", "-"}
		input := bufio.NewReader(bytes.NewBufferString(inputData))

		result, ok := tryProcFilter(args, input, func() {})
		assert.NotNil(t, result)
		assert.True(t, ok)

		output, err := io.ReadAll(result)
		require.NoError(t, err)
		assert.Equal(t, inputData, string(output))
	})

	t.Run("Filter fails with error", func(t *testing.T) {
		inputData := "input data"

		var cleanedUp atomic.Bool

		args := []string{"sh", "-c", "echo 'oh no' 1>&2; exit 21"}
		input := bufio.NewReader(bytes.NewBufferString(inputData))

		result, ok := tryProcFilter(args, input, func() { cleanedUp.Store(true) })
		assert.NotNil(t, result)
		assert.True(t, ok)

		_, err := io.ReadAll(result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "oh no: exit status 21")
		assert.Eventually(t, func() bool {
			return cleanedUp.Load()
		}, 5*time.Second, 10*time.Millisecond, "clean up function was not called")
	})
}
