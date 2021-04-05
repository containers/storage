package storage

import (
	"reflect"
	"testing"

	"github.com/containers/storage/pkg/idtools"
)

func TestGetAutoUserNSMapping(t *testing.T) {
	type args struct {
		size                  int
		availableUIDs         *idSet
		availableGIDs         *idSet
		usedUIDMappings       []idtools.IDMap
		usedGIDMappings       []idtools.IDMap
		additionalUIDMappings []idtools.IDMap
		additionalGIDMappings []idtools.IDMap
	}
	tests := []struct {
		name            string
		args            args
		wantUIDMappings []idtools.IDMap
		wantGIDMappings []idtools.IDMap
		wantErr         bool
	}{
		{
			name: "Normal",
			args: args{
				size:          65536,
				availableUIDs: newIDSet([]interval{{1000, 100000}}),
				availableGIDs: newIDSet([]interval{{1000, 100000}}),
			},
			wantUIDMappings: []idtools.IDMap{{ContainerID: 0, HostID: 1000, Size: 65536}},
			wantGIDMappings: []idtools.IDMap{{ContainerID: 0, HostID: 1000, Size: 65536}},
		},
		{
			name: "NotEnoughAvailableUIDs",
			args: args{
				size:          65536,
				availableUIDs: newIDSet([]interval{{1000, 10000}}),
				availableGIDs: newIDSet([]interval{{1000, 100000}}),
			},
			wantErr: true,
		},
		{
			name: "NotEnoughAvailableGIDs",
			args: args{
				size:          65536,
				availableUIDs: newIDSet([]interval{{1000, 100000}}),
				availableGIDs: newIDSet([]interval{{1000, 10000}}),
			},
			wantErr: true,
		},
		{
			name: "WithUsedIDs",
			args: args{
				size:            65536,
				availableUIDs:   newIDSet([]interval{{1000, 100000}}),
				availableGIDs:   newIDSet([]interval{{1000, 100000}}),
				usedUIDMappings: []idtools.IDMap{{ContainerID: 0, HostID: 2000, Size: 10000}},
				usedGIDMappings: []idtools.IDMap{
					{ContainerID: 0, HostID: 1000, Size: 10000},
					{ContainerID: 10000, HostID: 20000, Size: 5000},
					{ContainerID: 15000, HostID: 30000, Size: 5000},
				},
			},
			wantUIDMappings: []idtools.IDMap{
				{ContainerID: 0, HostID: 1000, Size: 1000},
				{ContainerID: 1000, HostID: 12000, Size: 65536 - 1000},
			},
			wantGIDMappings: []idtools.IDMap{
				{ContainerID: 0, HostID: 11000, Size: 9000},
				{ContainerID: 9000, HostID: 25000, Size: 5000},
				{ContainerID: 14000, HostID: 35000, Size: 65536 - 9000 - 5000},
			},
		},
		{
			name: "WithUsedAndAdditionalIDs",
			args: args{
				size:            65536,
				availableUIDs:   newIDSet([]interval{{1000, 100000}}),
				availableGIDs:   newIDSet([]interval{{1000, 100000}}),
				usedUIDMappings: []idtools.IDMap{{ContainerID: 0, HostID: 2000, Size: 10000}},
				usedGIDMappings: []idtools.IDMap{
					{ContainerID: 0, HostID: 1000, Size: 10000},
					{ContainerID: 10000, HostID: 20000, Size: 5000},
					{ContainerID: 15000, HostID: 30000, Size: 5000},
				},
				additionalUIDMappings: []idtools.IDMap{{ContainerID: 0, HostID: 1000, Size: 1}},
				additionalGIDMappings: []idtools.IDMap{{ContainerID: 1, HostID: 1001, Size: 1}},
			},
			wantUIDMappings: []idtools.IDMap{
				{ContainerID: 1, HostID: 1001, Size: 999},
				{ContainerID: 1000, HostID: 12000, Size: 65535 - 999},
				{ContainerID: 0, HostID: 1000, Size: 1},
			},
			wantGIDMappings: []idtools.IDMap{
				{ContainerID: 0, HostID: 11000, Size: 1},
				{ContainerID: 2, HostID: 11001, Size: 8999},
				{ContainerID: 9001, HostID: 25000, Size: 5000},
				{ContainerID: 14001, HostID: 35000, Size: 65535 - 1 - 8999 - 5000},
				{ContainerID: 1, HostID: 1001, Size: 1},
			},
		},
		{
			name: "DiscontinuedAvailableIntervals",
			args: args{
				size:          65536,
				availableUIDs: newIDSet([]interval{{1000, 50000}, {80000, 130000}}),
				availableGIDs: newIDSet([]interval{{1000, 30000}, {70000, 90000}, {110000, 160000}}),
				usedUIDMappings: []idtools.IDMap{
					{ContainerID: 0, HostID: 2000, Size: 10000},
					{ContainerID: 0, HostID: 10000, Size: 10000},
					{ContainerID: 0, HostID: 40000, Size: 10000},
				},
				usedGIDMappings: []idtools.IDMap{
					{ContainerID: 0, HostID: 1000, Size: 10000},
					{ContainerID: 100, HostID: 20000, Size: 5000},
					{ContainerID: 150, HostID: 30000, Size: 5000},
				},
				additionalUIDMappings: []idtools.IDMap{
					{ContainerID: 0, HostID: 1002, Size: 1},
				},
				additionalGIDMappings: []idtools.IDMap{
					{ContainerID: 0, HostID: 1003, Size: 1},
					{ContainerID: 1, HostID: 1001, Size: 1},
					{ContainerID: 2, HostID: 1006, Size: 1},
					{ContainerID: 100, HostID: 1100, Size: 10},
				},
			},
			wantUIDMappings: []idtools.IDMap{
				{ContainerID: 1, HostID: 1000, Size: 2},
				{ContainerID: 3, HostID: 1003, Size: 997},
				{ContainerID: 1000, HostID: 20000, Size: 20000},
				{ContainerID: 21000, HostID: 80000, Size: 65535 - 2 - 997 - 20000},
				{ContainerID: 0, HostID: 1002, Size: 1},
			},
			wantGIDMappings: []idtools.IDMap{
				{ContainerID: 3, HostID: 11000, Size: 97},
				{ContainerID: 110, HostID: 11000 + 97, Size: /*9000-97=*/ 8903},
				{ContainerID: /*110+8903=*/ 9013, HostID: 25000, Size: 5000},
				{ContainerID: /*9013+5000=*/ 14013, HostID: 70000, Size: 20000},
				{ContainerID: /*14013+20000*/ 34013, HostID: 110000, Size: 65536 - 13 - 97 - 8903 - 5000 - 20000},
				{ContainerID: 0, HostID: 1003, Size: 1},
				{ContainerID: 1, HostID: 1001, Size: 1},
				{ContainerID: 2, HostID: 1006, Size: 1},
				{ContainerID: 100, HostID: 1100, Size: 10},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUIDMappings, gotGIDMappings, err := getAutoUserNSIDMappings(
				tt.args.size,
				tt.args.availableUIDs, tt.args.availableGIDs,
				tt.args.usedUIDMappings, tt.args.usedGIDMappings,
				tt.args.additionalUIDMappings, tt.args.additionalGIDMappings,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAutoUserNSMapping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotUIDMappings, tt.wantUIDMappings) {
				t.Errorf("getAutoUserNSMapping() gotUIDMappings = %v, want %v", gotUIDMappings, tt.wantUIDMappings)
			}
			if !reflect.DeepEqual(gotGIDMappings, tt.wantGIDMappings) {
				t.Errorf("getAutoUserNSMapping() gotGIDMappings = %v, want %v", gotGIDMappings, tt.wantGIDMappings)
			}
		})
	}
}
