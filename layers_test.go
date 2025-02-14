package storage

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLayerLocationFromIndex(t *testing.T) {
	tests := []struct {
		index    int
		expected layerLocations
	}{
		{0, 1},
		{1, 2},
		{2, 4},
		{3, 8},
		{4, 16},
	}
	for _, test := range tests {
		result := layerLocationFromIndex(test.index)
		assert.Equal(t, test.expected, result)
	}
}

func TestLayerLocationFromIndexAndToIndex(t *testing.T) {
	var l layerLocations
	for i := 0; i < int(unsafe.Sizeof(l)*8); i++ {
		location := layerLocationFromIndex(i)
		index := indexFromLayerLocation(location)
		require.Equal(t, i, index)
	}
}
