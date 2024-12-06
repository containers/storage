package dedup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func wasVisited(d *dedupFiles, dev, ino uint64) bool {
	d.lock.Lock()
	defer d.lock.Unlock()

	di := deviceInodePair{
		dev: dev,
		ino: ino,
	}

	_, visited := d.visitedInodes[di]
	return visited
}

func TestRecordAndCheckInode(t *testing.T) {
	d, err := newDedupFiles()
	if err == notSupported {
		t.Skip("dedupFiles is not supported on this platform")
	}

	require.NoError(t, err)

	ino := uint64(1)
	firstDevice := uint64(100101)

	anotherIno := uint64(2)
	anotherDevice := uint64(100102)

	visited := wasVisited(d, firstDevice, ino)
	assert.False(t, visited)

	visited, err = d.recordInode(firstDevice, ino)
	assert.NoError(t, err)
	assert.False(t, visited)

	visited = wasVisited(d, firstDevice, ino)
	assert.True(t, visited)

	visited, err = d.recordInode(firstDevice, ino)
	assert.NoError(t, err)
	assert.True(t, visited)

	visited = wasVisited(d, firstDevice, anotherIno)
	assert.False(t, visited)

	visited = wasVisited(d, firstDevice, anotherIno)
	assert.False(t, visited)

	visited = wasVisited(d, anotherDevice, anotherIno)
	assert.False(t, visited)
}
