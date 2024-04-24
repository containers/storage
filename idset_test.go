package storage

import (
	"reflect"
	"testing"

	"github.com/containers/storage/pkg/idtools"
	"github.com/google/go-intervals/intervalset"
)

func allIntervals(s *idSet) []interval {
	iterator, cancel := s.iterator()
	defer cancel()
	var out []interval
	for i := iterator(); i != nil; i = iterator() {
		out = append(out, *i)
	}
	return out
}

func idSetsEqual(x, y *idSet) bool {
	return reflect.DeepEqual(allIntervals(x), allIntervals(y))
}

func TestNewIDSet(t *testing.T) {
	tests := []struct {
		name      string
		intervals []interval
		want      *idSet
	}{
		{
			"Nil",
			nil,
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		},
		{
			"Empty",
			[]interval{},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		},
		{
			"OneValidRange",
			[]interval{{3, 4}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{3, 4}})},
		},
		{
			"OneEmptyRange",
			[]interval{{3, 3}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		},
		{
			"OneInvalidRange",
			[]interval{{3, 2}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		},
		{
			"TwoRanges",
			[]interval{{1, 4}, {6, 8}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 4}, interval{6, 8}})},
		},
		{
			"SecondRangeEmpty",
			[]interval{{1, 4}, {6, 6}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 4}})},
		},
		{
			"SecondRangeInvalid",
			[]interval{{1, 4}, {6, 5}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 4}})},
		},
		{
			"TwoOverlappingRanges",
			[]interval{{1, 4}, {3, 6}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 6}})},
		},
		{
			"TwoAdjacentRanges",
			[]interval{{1, 4}, {4, 6}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 6}})},
		},
		{
			"TwoUnorderedRanges",
			[]interval{{6, 8}, {1, 4}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 4}, interval{6, 8}})},
		},
		{
			"ThreeRanges",
			[]interval{{1, 4}, {6, 8}, {11, 20}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 4}, interval{6, 8}, interval{11, 20}})},
		},
		{
			"ThreeRangesWithOverlap",
			[]interval{{1, 4}, {3, 8}, {11, 20}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 8}, interval{11, 20}})},
		},
		{
			"FourOverlappingRanges",
			[]interval{{1, 4}, {6, 8}, {11, 20}, {3, 30}},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 30}})},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newIDSet(tt.intervals); !idSetsEqual(got, tt.want) {
				t.Errorf("newIDSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHostAndContainerIDs(t *testing.T) {
	tests := []struct {
		name             string
		idMaps           []idtools.IDMap
		wantContainerIDs *idSet
		wantHostIDs      *idSet
	}{
		{
			"Nil",
			nil,
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		},
		{
			"Empty",
			[]idtools.IDMap{},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		},
		{
			"OneInvalidInterval",
			[]idtools.IDMap{
				{ContainerID: 10, HostID: 20, Size: -1},
			},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		},
		{
			"OneEmptyInterval",
			[]idtools.IDMap{
				{ContainerID: 10, HostID: 20, Size: -1},
			},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		},
		{
			"OneValidInterval",
			[]idtools.IDMap{
				{ContainerID: 10, HostID: 20, Size: 5},
			},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 15}})},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{20, 25}})},
		},
		{
			"TwoValidIntervals",
			[]idtools.IDMap{
				{ContainerID: 10, HostID: 20, Size: 5},
				{ContainerID: 30, HostID: 25, Size: 10},
			},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 15}, interval{30, 40}})},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{20, 35}})},
		},
		{
			"MultipleIntervals",
			[]idtools.IDMap{
				{ContainerID: 10, HostID: 20, Size: 10},
				{ContainerID: 30, HostID: 28, Size: 20},
				{ContainerID: 40, HostID: 30, Size: -1},
				{ContainerID: 45, HostID: 50, Size: 0},
				{ContainerID: 48, HostID: 60, Size: 20},
			},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 20}, interval{30, 68}})},
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{20, 48}, interval{60, 80}})},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHostIDs(tt.idMaps); !idSetsEqual(got, tt.wantHostIDs) {
				t.Errorf("getHostIDs() = %v, want %v", got, tt.wantHostIDs)
			}
			if got := getContainerIDs(tt.idMaps); !idSetsEqual(got, tt.wantContainerIDs) {
				t.Errorf("getContainerIDs() = %v, want %v", got, tt.wantContainerIDs)
			}
		})
	}
}

var idSetOperatorsTestCases = []struct {
	name         string
	x            *idSet
	y            *idSet
	subtractWant *idSet
	unionWant    *idSet
	zipWant      []idtools.IDMap
}{
	{
		"NilAndNil",
		nil,
		nil,
		nil,
		nil,
		([]idtools.IDMap)(nil),
	},
	{
		"NilAndNotNil",
		nil,
		newIDSet([]interval{{1, 5}}),
		nil,
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		([]idtools.IDMap)(nil),
	},
	{
		"NotNilAndNil",
		newIDSet([]interval{{1, 5}}),
		nil,
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		([]idtools.IDMap)(nil),
	},
	{
		"EmptyAndEmpty",
		newIDSet([]interval{}),
		newIDSet([]interval{}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		&idSet{set: intervalset.NewImmutableSet(nil)},
		([]idtools.IDMap)(nil),
	},
	{
		"EmptyAndNotEmpty",
		newIDSet([]interval{}),
		newIDSet([]interval{{1, 5}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		([]idtools.IDMap)(nil),
	},
	{
		"NotEmptyAndEmpty",
		newIDSet([]interval{{1, 5}}),
		newIDSet([]interval{}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		([]idtools.IDMap)(nil),
	},
	{
		"OneIntervalAndSameInterval",
		newIDSet([]interval{{1, 5}}),
		newIDSet([]interval{{1, 5}}),
		&idSet{set: intervalset.NewImmutableSet(nil)},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		[]idtools.IDMap{{ContainerID: 1, HostID: 1, Size: 4}},
	},
	{
		"OneIntervalAndSingleSmallerInterval",
		newIDSet([]interval{{1, 5}}),
		newIDSet([]interval{{3, 4}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 3}, interval{4, 5}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		[]idtools.IDMap{{ContainerID: 3, HostID: 1, Size: 1}},
	},
	{
		"OneIntervalAndSingleBiggerInterval",
		newIDSet([]interval{{1, 5}}),
		newIDSet([]interval{{0, 8}}),
		&idSet{set: intervalset.NewImmutableSet(nil)},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{0, 8}})},
		[]idtools.IDMap{{ContainerID: 0, HostID: 1, Size: 4}},
	},
	{
		"OneIntervalAndSingleIntervalLeft",
		newIDSet([]interval{{1, 5}}),
		newIDSet([]interval{{1, 3}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{3, 5}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		[]idtools.IDMap{{ContainerID: 1, HostID: 1, Size: 2}},
	},
	{
		"OneIntervalAndSingleIntervalRight",
		newIDSet([]interval{{1, 5}}),
		newIDSet([]interval{{2, 5}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 2}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}})},
		[]idtools.IDMap{{ContainerID: 2, HostID: 1, Size: 3}},
	},
	{
		"OneIntervalAndSingleIntervalLeftAndBeyond",
		newIDSet([]interval{{1, 5}}),
		newIDSet([]interval{{-2, 3}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{3, 5}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{-2, 5}})},
		[]idtools.IDMap{{ContainerID: -2, HostID: 1, Size: 4}},
	},
	{
		"OneIntervalAndSingleIntervalRightAndBeyond",
		newIDSet([]interval{{1, 5}}),
		newIDSet([]interval{{2, 8}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 2}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 8}})},
		[]idtools.IDMap{{ContainerID: 2, HostID: 1, Size: 4}},
	},
	{
		"OneIntervalAndSingleIntervalAdjacentLeft",
		newIDSet([]interval{{10, 20}}),
		newIDSet([]interval{{2, 10}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 20}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{2, 20}})},
		[]idtools.IDMap{{ContainerID: 2, HostID: 10, Size: 8}},
	},
	{
		"OneIntervalAndSingleIntervalAdjacentRight",
		newIDSet([]interval{{10, 20}}),
		newIDSet([]interval{{20, 30}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 20}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 30}})},
		[]idtools.IDMap{{ContainerID: 20, HostID: 10, Size: 10}},
	},
	{
		"OneIntervalAndSingleIntervalNonOverlapLeft",
		newIDSet([]interval{{10, 20}}),
		newIDSet([]interval{{2, 5}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 20}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{2, 5}, interval{10, 20}})},
		[]idtools.IDMap{{ContainerID: 2, HostID: 10, Size: 3}},
	},
	{
		"OneIntervalAndSingleIntervalNonOverlapRight",
		newIDSet([]interval{{10, 20}}),
		newIDSet([]interval{{25, 30}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 20}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 20}, interval{25, 30}})},
		[]idtools.IDMap{{ContainerID: 25, HostID: 10, Size: 5}},
	},
	{
		"OneIntervalAndMultipleIntervals",
		newIDSet([]interval{{10, 100}}),
		newIDSet([]interval{
			{2, 5},
			{8, 10},
			{20, 30},
			{40, 50},
			{80, 120},
		}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{10, 20}, interval{30, 40}, interval{50, 80}})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{2, 5}, interval{8, 120}})},
		[]idtools.IDMap{
			{ContainerID: 2, HostID: 10, Size: 3},
			{ContainerID: 8, HostID: 13, Size: 2},
			{ContainerID: 20, HostID: 15, Size: 10},
			{ContainerID: 40, HostID: 25, Size: 10},
			{ContainerID: 80, HostID: 35, Size: 40},
		},
	},
	{
		"MultipleIntervalsAndSingle",
		newIDSet([]interval{
			{2, 5},
			{8, 10},
			{20, 30},
			{40, 50},
			{80, 120},
		}),
		newIDSet([]interval{{10, 45}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{
			interval{2, 5},
			interval{8, 10},
			interval{45, 50},
			interval{80, 120},
		})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{2, 5}, interval{8, 50}, interval{80, 120}})},
		[]idtools.IDMap{
			{ContainerID: 10, HostID: 2, Size: 3},
			{ContainerID: 13, HostID: 8, Size: 2},
			{ContainerID: 15, HostID: 20, Size: 10},
			{ContainerID: 25, HostID: 40, Size: 10},
			{ContainerID: 35, HostID: 80, Size: 10},
		},
	},
	{
		"MultipleIntervalsAndMultipleIntervals",
		newIDSet([]interval{
			{2, 5},
			{8, 10},
			{20, 30},
			{40, 50},
			{80, 120},
		}),
		newIDSet([]interval{{10, 45}, {90, 100}, {130, 150}}),
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{
			interval{2, 5},
			interval{8, 10},
			interval{45, 50},
			interval{80, 90},
			interval{100, 120},
		})},
		&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{
			interval{2, 5},
			interval{8, 50},
			interval{80, 120},
			interval{130, 150},
		})},
		[]idtools.IDMap{
			{ContainerID: 10, HostID: 2, Size: 3},
			{ContainerID: 13, HostID: 8, Size: 2},
			{ContainerID: 15, HostID: 20, Size: 10},
			{ContainerID: 25, HostID: 40, Size: 10},
			{ContainerID: 35, HostID: 80, Size: 10},
			{ContainerID: 90, HostID: 90, Size: 10},
			{ContainerID: 130, HostID: 100, Size: 20},
		},
	},
}

func TestIDSetSubtract(t *testing.T) {
	for _, tt := range idSetOperatorsTestCases {
		t.Run(tt.name, func(t *testing.T) {
			var ix, iy []interval
			if tt.x != nil {
				ix = allIntervals(tt.x)
			}
			if tt.y != nil {
				iy = allIntervals(tt.y)
			}
			if got := tt.x.subtract(tt.y); !idSetsEqual(got, tt.subtractWant) {
				t.Errorf("idSet.subtract() = %v, want %v", got, tt.subtractWant)
			}
			// Make sure x and y are unchanged.
			if tt.x != nil {
				if jx := allIntervals(tt.x); !reflect.DeepEqual(ix, jx) {
					t.Errorf("x changed from %v to %v", ix, jx)
				}
			}
			if tt.y != nil {
				if jy := allIntervals(tt.y); !reflect.DeepEqual(iy, jy) {
					t.Errorf("y changed from %v to %v", iy, jy)
				}
			}
		})
	}
}

func TestIDSetUnion(t *testing.T) {
	for _, tt := range idSetOperatorsTestCases {
		t.Run(tt.name, func(t *testing.T) {
			var ix, iy []interval
			if tt.x != nil {
				ix = allIntervals(tt.x)
			}
			if tt.y != nil {
				iy = allIntervals(tt.y)
			}
			if got := tt.x.union(tt.y); !idSetsEqual(got, tt.unionWant) {
				t.Errorf("idSet.union() = %v, want %v", got, tt.unionWant)
			}
			// Make sure x and y are unchanged.
			if tt.x != nil {
				if jx := allIntervals(tt.x); !reflect.DeepEqual(ix, jx) {
					t.Errorf("x changed from %v to %v", ix, jx)
				}
			}
			if tt.y != nil {
				if jy := allIntervals(tt.y); !reflect.DeepEqual(iy, jy) {
					t.Errorf("y changed from %v to %v", iy, jy)
				}
			}
		})
	}
}

func TestIDSetSize(t *testing.T) {
	tests := []struct {
		name string
		set  *idSet
		want int
	}{
		{"Nil", nil, 0},
		{"Empty", newIDSet([]interval{}), 0},
		{"EmptyInterval", newIDSet([]interval{{3, 3}}), 0},
		{"InvalidInterval", newIDSet([]interval{{5, 3}}), 0},
		{"OneInterval", newIDSet([]interval{{3, 9}}), 6},
		{"TwoIntervals", newIDSet([]interval{{3, 9}, {20, 25}}), 11},
		{"TwoAdjacentIntervals", newIDSet([]interval{{3, 9}, {9, 20}}), 17},
		{"TwoOverlappingIntervals", newIDSet([]interval{{3, 9}, {6, 20}}), 17},
		{"TwoContainingIntervals", newIDSet([]interval{{3, 9}, {0, 20}}), 20},
		{"ThreeIntervals", newIDSet([]interval{{3, 9}, {15, 20}, {30, 40}}), 21},
		{"MultipleIntervals", newIDSet([]interval{{3, 9}, {15, 20}, {3, 0}, {6, 12}, {30, 40}}), 24},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.size(); got != tt.want {
				t.Errorf("idSet.size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIDSetFindAvailable(t *testing.T) {
	tests := []struct {
		name    string
		set     *idSet
		n       int
		want    *idSet
		wantErr bool
	}{
		{
			"NilIntervalFindZero",
			nil,
			0,
			&idSet{set: intervalset.NewImmutableSet(nil)},
			false,
		},
		{
			"NilIntervalFindPositive",
			nil,
			1,
			nil,
			true,
		},
		{
			"NilIntervalFindNegative",
			nil,
			-1,
			&idSet{set: intervalset.NewImmutableSet(nil)},
			false,
		},
		{
			"EmptyIntervalFindZero",
			newIDSet([]interval{}),
			0,
			&idSet{set: intervalset.NewImmutableSet(nil)},
			false,
		},
		{
			"EmptyIntervalFindPositive",
			newIDSet([]interval{}),
			1,
			nil,
			true,
		},
		{
			"EmptyIntervalFindNegative",
			newIDSet([]interval{}),
			-1,
			&idSet{set: intervalset.NewImmutableSet(nil)},
			false,
		},
		{
			"OneIntervalFindOK",
			newIDSet([]interval{{1, 5}}),
			3,
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 4}})},
			false,
		},
		{
			"OneIntervalFindNotEnough",
			newIDSet([]interval{{1, 5}}),
			5,
			nil,
			true,
		},
		{
			"OneIntervalFindZero",
			newIDSet([]interval{{1, 5}}),
			0,
			&idSet{set: intervalset.NewImmutableSet(nil)},
			false,
		},
		{
			"OneIntervalFindNegative",
			newIDSet([]interval{{1, 5}}),
			-1,
			&idSet{set: intervalset.NewImmutableSet(nil)},
			false,
		},
		{
			"MultipleIntervalsFindOK",
			newIDSet([]interval{{1, 5}, {9, 15}, {20, 30}}),
			15,
			&idSet{set: intervalset.NewImmutableSet([]intervalset.Interval{interval{1, 5}, interval{9, 15}, interval{20, 25}})},
			false,
		},
		{
			"MultipleIntervalsFindNotEnough",
			newIDSet([]interval{{1, 5}, {9, 15}, {20, 30}}),
			25,
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.set.findAvailable(tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("idSet.findAvailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !idSetsEqual(got, tt.want) {
				t.Errorf("idSet.findAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIDSetZip(t *testing.T) {
	for _, tt := range idSetOperatorsTestCases {
		t.Run(tt.name, func(t *testing.T) {
			var ix, iy []interval
			if tt.x != nil {
				ix = allIntervals(tt.x)
			}
			if tt.y != nil {
				iy = allIntervals(tt.y)
			}
			if got := tt.x.zip(tt.y); !reflect.DeepEqual(got, tt.zipWant) {
				t.Errorf("idSet.zip() = %v, want %v", got, tt.unionWant)
			}
			// Make sure x and y are unchanged.
			if tt.x != nil {
				if jx := allIntervals(tt.x); !reflect.DeepEqual(ix, jx) {
					t.Errorf("x changed from %v to %v", ix, jx)
				}
			}
			if tt.y != nil {
				if jy := allIntervals(tt.y); !reflect.DeepEqual(iy, jy) {
					t.Errorf("y changed from %v to %v", iy, jy)
				}
			}
		})
	}
}

func TestIntervalLength(t *testing.T) {
	tests := []struct {
		name  string
		start int
		end   int
		want  int
	}{
		{"ZeroLength", 3, 3, 0},
		{"PositiveLength", 2, 10, 8},
		{"NegativeLength", 10, 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := interval{
				start: tt.start,
				end:   tt.end,
			}
			if got := i.length(); got != tt.want {
				t.Errorf("interval.length() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntervalIsZero(t *testing.T) {
	tests := []struct {
		name  string
		start int
		end   int
		want  bool
	}{
		{"ZeroLength", 3, 3, true},
		{"PositiveLength", 2, 10, false},
		{"NegativeLength", 10, 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := interval{
				start: tt.start,
				end:   tt.end,
			}
			if got := i.IsZero(); got != tt.want {
				t.Errorf("interval.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

// assertIntervalSame asserts `got` equals to `want` considering zero check. If the wanted interval
// is empty, we only want to assert IsZero() == true, instead of the exact number.
func assertIntervalSame(t *testing.T, got intervalset.Interval, want *interval, name string) {
	t.Helper()
	if want == nil && !got.IsZero() {
		t.Errorf("%v = %v, want nil", name, got)
	} else if want != nil && !reflect.DeepEqual(got, *want) {
		t.Errorf("%v = %v, want %v", name, got, *want)
	}
}

var intervalTestCases = []struct {
	name                             string
	start, end                       int
	otherStart, otherEnd             int
	intersectWant                    *interval
	beforeWant, reflectiveBeforeWant bool
	bisectWant, reflectiveBisectWant [2]*interval
	adjoinWant                       *interval
	encompassWant                    *interval
}{
	{
		"TwoZeroIntervals", 0, 0, 0, 0,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{nil, nil},
		nil,
		nil,
	},
	{
		"ZeroIntervalAndInvalidInterval", 0, 0, 5, 3,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{nil, nil},
		nil,
		nil,
	},
	{
		"ZeroIntervalAndInvalidIntervalIncludingZero", 0, 0, 5, -3,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{nil, nil},
		nil,
		nil,
	},
	{
		"TwoIncludingInvalidIntervals", 3, -2, 5, -3,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{nil, nil},
		nil,
		nil,
	},
	{
		"TwoInvalidIntervals", 8, 6, 5, -3,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{nil, nil},
		nil,
		nil,
	},
	{
		"ZeroIntervalAndNormalInterval", 0, 0, 3, 5,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{{3, 5}, nil},
		nil,
		&interval{3, 5},
	},
	{
		"ZeroIntervalAndNormalIntervalIncludingZero", 0, 0, -3, 5,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{{-3, 5}, nil},
		nil,
		&interval{-3, 5},
	},
	{
		"InvalidIntervalAndNormalInterval", 5, 3, 6, 8,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{{6, 8}, nil},
		nil,
		&interval{6, 8},
	},
	{
		"InvalidIntervalAndNormalIntervalEnclosing", 5, 3, 2, 6,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{{2, 6}, nil},
		nil,
		&interval{2, 6},
	},
	{
		"InvalidIntervalAndNormalIntervalIntersecting", 5, 3, 4, 6,
		nil,
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{{4, 6}, nil},
		nil,
		&interval{4, 6},
	},
	{
		"IntersectingIntervals", 5, 8, 6, 9,
		&interval{6, 8},
		false, false,
		[2]*interval{{5, 6}, nil},
		[2]*interval{nil, {8, 9}},
		nil,
		&interval{5, 9},
	},
	{
		"AdjoinIntervals", 5, 8, 8, 12,
		nil,
		false, false,
		[2]*interval{{5, 8}, nil},
		[2]*interval{nil, {8, 12}},
		&interval{5, 12},
		&interval{5, 12},
	},
	{
		"EnclosingIntervals", 5, 8, 4, 12,
		&interval{5, 8},
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{{4, 5}, {8, 12}},
		nil,
		&interval{4, 12},
	},
	{
		"EnclosingIntervalsMeetOneEnd", 5, 8, 5, 12,
		&interval{5, 8},
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{nil, {8, 12}},
		nil,
		&interval{5, 12},
	},
	{
		"EqualIntervals", 5, 8, 5, 8,
		&interval{5, 8},
		false, false,
		[2]*interval{nil, nil},
		[2]*interval{nil, nil},
		nil,
		&interval{5, 8},
	},
	{
		"NonIntersectingIntervals", 3, 5, 8, 10,
		nil,
		true, false,
		[2]*interval{{3, 5}, nil},
		[2]*interval{nil, {8, 10}},
		nil,
		&interval{3, 10},
	},
}

func TestIntervalIntersect(t *testing.T) {
	for _, tt := range intervalTestCases {
		i := interval{tt.start, tt.end}
		j := interval{tt.otherStart, tt.otherEnd}
		t.Run(tt.name, func(t *testing.T) {
			assertIntervalSame(t, i.Intersect(j), tt.intersectWant, "interval.Intersect()")
		})
		t.Run("Reflective_"+tt.name, func(t *testing.T) {
			assertIntervalSame(t, j.Intersect(i), tt.intersectWant, "interval.Intersect()")
		})
	}
}

func TestIntervalBefore(t *testing.T) {
	for _, tt := range intervalTestCases {
		i := interval{tt.start, tt.end}
		j := interval{tt.otherStart, tt.otherEnd}
		t.Run(tt.name, func(t *testing.T) {
			if got := i.Before(j); got != tt.beforeWant {
				t.Errorf("interval.Before() = %v, want %v", got, tt.beforeWant)
			}
		})
		t.Run("Reflective_"+tt.name, func(t *testing.T) {
			if got := j.Before(i); got != tt.reflectiveBeforeWant {
				t.Errorf("interval.Before() = %v, want %v", got, tt.reflectiveBeforeWant)
			}
		})
	}
}

func TestIntervalBisect(t *testing.T) {
	for _, tt := range intervalTestCases {
		i := interval{tt.start, tt.end}
		j := interval{tt.otherStart, tt.otherEnd}
		t.Run(tt.name, func(t *testing.T) {
			x, y := i.Bisect(j)
			assertIntervalSame(t, x, tt.bisectWant[0], "interval.Bisect()[0]")
			assertIntervalSame(t, y, tt.bisectWant[1], "interval.Bisect()[1]")
		})
		t.Run("Reflective_"+tt.name, func(t *testing.T) {
			x, y := j.Bisect(i)
			assertIntervalSame(t, x, tt.reflectiveBisectWant[0], "interval.Bisect()[0]")
			assertIntervalSame(t, y, tt.reflectiveBisectWant[1], "interval.Bisect()[1]")
		})
	}
}

func TestIntervalAdjoin(t *testing.T) {
	for _, tt := range intervalTestCases {
		i := interval{tt.start, tt.end}
		j := interval{tt.otherStart, tt.otherEnd}
		t.Run(tt.name, func(t *testing.T) {
			assertIntervalSame(t, i.Adjoin(j), tt.adjoinWant, "interval.Adjoin()")
		})
		t.Run("Reflective_"+tt.name, func(t *testing.T) {
			assertIntervalSame(t, j.Adjoin(i), tt.adjoinWant, "interval.Adjoin()")
		})
	}
}

func TestIntervalEncompass(t *testing.T) {
	for _, tt := range intervalTestCases {
		i := interval{tt.start, tt.end}
		j := interval{tt.otherStart, tt.otherEnd}
		t.Run(tt.name, func(t *testing.T) {
			assertIntervalSame(t, i.Encompass(j), tt.encompassWant, "interval.Encompass()")
		})
		t.Run("Reflective_"+tt.name, func(t *testing.T) {
			assertIntervalSame(t, j.Encompass(i), tt.encompassWant, "interval.Encompass()")
		})
	}
}

func TestOverlappingMappings(t *testing.T) {
	mappings := []idtools.IDMap{{ContainerID: 0, HostID: 1000, Size: 65536}, {ContainerID: 0, HostID: 1000, Size: 65536}}
	if err := hasOverlappingRanges(mappings); err == nil {
		t.Errorf("mappings = %v, expected to be overlapping", mappings)
	}

	mappings = []idtools.IDMap{{ContainerID: 0, HostID: 1000, Size: 65536}, {ContainerID: 65536, HostID: 5000, Size: 65536}}
	if err := hasOverlappingRanges(mappings); err == nil {
		t.Errorf("mappings = %v, expected to be overlapping", mappings)
	}

	mappings = []idtools.IDMap{{ContainerID: 0, HostID: 1000, Size: 65536}, {ContainerID: 0, HostID: 5000 + 65536, Size: 65536}}
	if err := hasOverlappingRanges(mappings); err == nil {
		t.Errorf("mappings = %v, expected to be overlapping", mappings)
	}

	mappings = []idtools.IDMap{{ContainerID: 0, HostID: 1000, Size: 65536}, {ContainerID: 0, HostID: 1000 + 65536, Size: 65536}}
	if err := hasOverlappingRanges(mappings); err == nil {
		t.Errorf("mappings = %v, expected to be overlapping", mappings)
	}

	mappings = []idtools.IDMap{{ContainerID: 0, HostID: 0, Size: 65536}, {ContainerID: 65536, HostID: 65536, Size: 65536}}
	if err := hasOverlappingRanges(mappings); err != nil {
		t.Errorf("mappings = %v, expected to not be overlapping", mappings)
	}

	mappings = []idtools.IDMap{{ContainerID: 0, HostID: 0, Size: 65536}, {ContainerID: 1 * 65536, HostID: 1 * 65536, Size: 65536}, {ContainerID: 2 * 65536, HostID: 2 * 65536, Size: 65536}}
	if err := hasOverlappingRanges(mappings); err != nil {
		t.Errorf("mappings = %v, expected to not be overlapping", mappings)
	}
}
