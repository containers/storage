package storage

import (
	"testing"

	"github.com/containers/storage/pkg/idtools"
	"github.com/stretchr/testify/assert"
)

func TestSubtractHostID(t *testing.T) {
	avail := idtools.IDMap{
		ContainerID: 0,
		HostID:      100000,
		Size:        65536,
	}
	used := idtools.IDMap{
		ContainerID: 0,
		HostID:      200000,
		Size:        65536,
	}
	ret := subtractHostIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, avail.HostID, ret[0].HostID)
	assert.Equal(t, avail.Size, ret[0].Size)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      0,
		Size:        65536,
	}
	ret = subtractHostIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, avail.HostID, ret[0].HostID)
	assert.Equal(t, avail.Size, ret[0].Size)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      0,
		Size:        65536,
	}
	ret = subtractHostIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, avail.HostID, ret[0].HostID)
	assert.Equal(t, avail.Size, ret[0].Size)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      100000,
		Size:        4096,
	}
	ret = subtractHostIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, ret[0].HostID, avail.HostID+4096)
	assert.Equal(t, ret[0].Size, avail.Size-4096)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      165536 - 4096,
		Size:        4096,
	}
	ret = subtractHostIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, ret[0].HostID, avail.HostID)
	assert.Equal(t, ret[0].Size, avail.Size-4096)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      132768,
		Size:        4096,
	}
	ret = subtractHostIDs(avail, used)
	assert.Equal(t, len(ret), 2)
	assert.Equal(t, ret[0].HostID, avail.HostID)
	assert.Equal(t, ret[0].Size, 32768)
	assert.Equal(t, ret[1].HostID, avail.HostID+32768+4096)
	assert.Equal(t, ret[1].Size, 32768-4096)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      132768,
		Size:        65536,
	}
	ret = subtractHostIDs(avail, used)
	assert.NotNil(t, ret)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      100000,
		Size:        65536,
	}
	ret = subtractHostIDs(avail, used)
	assert.Equal(t, len(ret), 0)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      100000,
		Size:        1000000,
	}
	ret = subtractHostIDs(avail, used)
	assert.Equal(t, len(ret), 0)
}

func TestSubtractContainerID(t *testing.T) {
	avail := idtools.IDMap{
		ContainerID: 100000,
		HostID:      0,
		Size:        65536,
	}
	used := idtools.IDMap{
		ContainerID: 200000,
		HostID:      0,
		Size:        65536,
	}
	ret := subtractContainerIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, avail.ContainerID, ret[0].ContainerID)
	assert.Equal(t, 0, ret[0].HostID)
	assert.Equal(t, avail.Size, ret[0].Size)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      0,
		Size:        65536,
	}
	ret = subtractContainerIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, avail.ContainerID, ret[0].ContainerID)
	assert.Equal(t, avail.Size, ret[0].Size)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      1,
		Size:        65536,
	}
	ret = subtractContainerIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, avail.ContainerID, ret[0].ContainerID)
	assert.Equal(t, 0, ret[0].HostID)
	assert.Equal(t, avail.Size, ret[0].Size)

	used = idtools.IDMap{
		ContainerID: 100000,
		HostID:      0,
		Size:        4096,
	}
	ret = subtractContainerIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, ret[0].ContainerID, avail.ContainerID+4096)
	assert.Equal(t, 4096, ret[0].HostID)
	assert.Equal(t, ret[0].Size, avail.Size-4096)

	used = idtools.IDMap{
		ContainerID: 165536 - 4096,
		HostID:      0,
		Size:        4096,
	}
	ret = subtractContainerIDs(avail, used)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, ret[0].ContainerID, avail.ContainerID)
	assert.Equal(t, ret[0].Size, avail.Size-4096)

	used = idtools.IDMap{
		ContainerID: 132768,
		HostID:      0,
		Size:        4096,
	}
	ret = subtractContainerIDs(avail, used)
	assert.Equal(t, len(ret), 2)
	assert.Equal(t, ret[0].ContainerID, avail.ContainerID)
	assert.Equal(t, ret[0].Size, 32768)
	assert.Equal(t, ret[1].ContainerID, 132768+4096)
	assert.Equal(t, ret[1].Size, 32768-4096)

	used = idtools.IDMap{
		ContainerID: 0,
		HostID:      132768,
		Size:        65536,
	}
	ret = subtractContainerIDs(avail, used)
	assert.NotNil(t, ret)

	used = idtools.IDMap{
		ContainerID: 100000,
		HostID:      0,
		Size:        65536,
	}
	ret = subtractContainerIDs(avail, used)
	assert.Equal(t, len(ret), 0)

	used = idtools.IDMap{
		ContainerID: 100000,
		HostID:      0,
		Size:        1000000,
	}
	ret = subtractContainerIDs(avail, used)
	assert.Equal(t, len(ret), 0)

	avail = idtools.IDMap{
		ContainerID: 0,
		HostID:      1000,
		Size:        65536,
	}
	used = idtools.IDMap{
		ContainerID: 10,
		HostID:      10,
		Size:        1,
	}
	ret = subtractContainerIDs(avail, used)
	assert.Equal(t, len(ret), 2)
	assert.Equal(t, ret[0].ContainerID, 0)
	assert.Equal(t, ret[0].HostID, 1000)
	assert.Equal(t, ret[0].Size, 10)
	assert.Equal(t, ret[1].ContainerID, 11)
	assert.Equal(t, ret[1].HostID, 1011)
	assert.Equal(t, ret[1].Size, 65525)
}

func TestFindAvailableIDRange(t *testing.T) {
	avail := []idtools.IDMap{
		{
			ContainerID: 0,
			HostID:      100000,
			Size:        65536,
		},
	}
	used := []idtools.IDMap{
		{
			ContainerID: 0,
			HostID:      100000,
			Size:        65536,
		},
	}
	_, err := findAvailableIDRange(4096, avail, used)
	assert.Error(t, err, "could not find enough available IDs")

	used = []idtools.IDMap{
		{
			ContainerID: 0,
			HostID:      100000,
			Size:        4096,
		},
	}
	_, err = findAvailableIDRange(100000, avail, used)
	assert.Error(t, err, "could not find enough available IDs")

	used = []idtools.IDMap{
		{
			ContainerID: 0,
			HostID:      100000,
			Size:        32768,
		},
	}

	ret, err := findAvailableIDRange(4096, avail, used)
	assert.Nil(t, err)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, ret[0].HostID, 100000+32768)
	assert.Equal(t, ret[0].Size, 4096)

	used = []idtools.IDMap{
		{
			ContainerID: 0,
			HostID:      100010,
			Size:        10,
		},
	}
	ret, err = findAvailableIDRange(4096, avail, used)
	assert.Nil(t, err)
	assert.Equal(t, len(ret), 2)
	assert.Equal(t, ret[0].HostID, 100000)
	assert.Equal(t, ret[1].HostID, 100020)
}
