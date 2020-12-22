package idtools

import (
	"testing"
)

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
