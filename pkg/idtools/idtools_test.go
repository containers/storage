package idtools

import (
	"testing"
)

func TestToHost(t *testing.T) {
	idMappings := []IDMap{
		{
			ContainerID: 0,
			HostID:      1000,
			Size:        1,
		},
		{
			ContainerID: 1,
			HostID:      100000,
			Size:        65536,
		},
	}

	mappings := IDMappings{
		uids: idMappings,
		gids: idMappings,
	}

	pair, err := mappings.ToHost(IDPair{UID: 0, GID: 0})
	if err != nil {
		t.Fatal(err)
	}
	if pair.UID != 1000 {
		t.Fatalf("Converted to the wrong UID")
	}
	if pair.GID != 1000 {
		t.Fatalf("Converted to the wrong GID")
	}

	pair, err = mappings.ToHost(IDPair{UID: 1000, GID: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if pair.UID != 100999 {
		t.Fatalf("Converted to the wrong UID")
	}
	if pair.GID != 100999 {
		t.Fatalf("Converted to the wrong GID")
	}
}

func TestToHostOverflow(t *testing.T) {
	idMappings := []IDMap{
		{
			ContainerID: 0,
			HostID:      1000,
			Size:        1,
		},
		{
			ContainerID: 1,
			HostID:      100000,
			Size:        65536,
		},
	}

	mappings := IDMappings{
		uids: idMappings,
		gids: idMappings,
	}

	pair, err := mappings.ToHostOverflow(IDPair{UID: 65538, GID: 0})
	if err != nil {
		t.Fatal(err)
	}
	if pair.UID != getOverflowUID() {
		t.Fatalf("Converted to the wrong UID")
	}
	if pair.GID != 1000 {
		t.Fatalf("Converted to the wrong GID")
	}

	pair, err = mappings.ToHostOverflow(IDPair{UID: 10, GID: 65539})
	if err != nil {
		t.Fatal(err)
	}
	if pair.UID != 100009 {
		t.Fatalf("Converted to the wrong UID")
	}
	if pair.GID != getOverflowGID() {
		t.Fatalf("Converted to the wrong GID")
	}
}

func TestGetRootUIDGID(t *testing.T) {
	mappingsUIDs := []IDMap{
		{
			ContainerID: 0,
			HostID:      1000,
			Size:        100,
		},
	}
	mappingsGIDs := []IDMap{
		{
			ContainerID: 0,
			HostID:      2000,
			Size:        100,
		},
	}
	uid, gid, err := GetRootUIDGID(mappingsUIDs, mappingsGIDs)
	if err != nil {
		t.Fatal(err)
	}
	if uid != 1000 {
		t.Fatalf("Detected wrong root uid in the host")
	}
	if gid != 2000 {
		t.Fatalf("Detected wrong root uid in the host")
	}

	mappingsUIDs = []IDMap{
		{
			ContainerID: 100,
			HostID:      1001,
			Size:        1,
		},
	}
	mappingsGIDs = []IDMap{
		{
			ContainerID: 200,
			HostID:      2002,
			Size:        1,
		},
	}
	uid, gid, err = GetRootUIDGID(mappingsUIDs, mappingsGIDs)
	if err != nil {
		t.Fatal(err)
	}
	if uid != 1001 {
		t.Fatalf("Detected wrong root uid in the host")
	}
	if gid != 2002 {
		t.Fatalf("Detected wrong root uid in the host")
	}

	mappingsUIDs = []IDMap{
		{
			ContainerID: 100,
			HostID:      1001,
			Size:        100,
		},
	}
	mappingsGIDs = []IDMap{
		{
			ContainerID: 200,
			HostID:      2002,
			Size:        100,
		},
	}
	_, _, err = GetRootUIDGID(mappingsUIDs, mappingsGIDs)
	if err == nil {
		t.Fatalf("Detected root user")
	}

	mappingsUIDs = []IDMap{
		{
			ContainerID: 100,
			HostID:      1001,
			Size:        1,
		},
		{
			ContainerID: 200,
			HostID:      2001,
			Size:        1,
		},
	}
	mappingsGIDs = []IDMap{
		{
			ContainerID: 100,
			HostID:      1001,
			Size:        1,
		},
		{
			ContainerID: 200,
			HostID:      2001,
			Size:        1,
		},
	}
	_, _, err = GetRootUIDGID(mappingsUIDs, mappingsGIDs)
	if err == nil {
		t.Fatalf("Detected root user")
	}
}

func TestIsContiguous(t *testing.T) {
	mappings := []IDMap{
		{
			ContainerID: 0,
			HostID:      0,
			Size:        100,
		},
		{
			ContainerID: 100,
			HostID:      100,
			Size:        100,
		},
	}
	if !IsContiguous(mappings) {
		t.Errorf("mappings %v expected to be contiguous", mappings)
	}
	mappings = []IDMap{
		{
			ContainerID: 0,
			HostID:      10000,
			Size:        100,
		},
		{
			ContainerID: 100,
			HostID:      100,
			Size:        100,
		},
	}
	if IsContiguous(mappings) {
		t.Errorf("mappings %v expected to not be contiguous", mappings)
	}

	mappings = []IDMap{
		{
			ContainerID: 10000,
			HostID:      0,
			Size:        100,
		},
		{
			ContainerID: 100,
			HostID:      100,
			Size:        100,
		},
	}
	if IsContiguous(mappings) {
		t.Errorf("mappings %v expected to not be contiguous", mappings)
	}

	mappings = []IDMap{
		{
			ContainerID: 0,
			HostID:      10,
			Size:        10,
		},
		{
			ContainerID: 10,
			HostID:      20,
			Size:        10,
		},
		{
			ContainerID: 20,
			HostID:      30,
			Size:        10,
		},
		{
			ContainerID: 30,
			HostID:      40,
			Size:        10,
		},
	}
	if !IsContiguous(mappings) {
		t.Errorf("mappings %v expected to be contiguous", mappings)
	}

	mappings = []IDMap{
		{
			ContainerID: 0,
			HostID:      10,
			Size:        10,
		},
	}
	if !IsContiguous(mappings) {
		t.Errorf("mappings %v expected to be contiguous", mappings)
	}
}
