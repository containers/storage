//go:build !windows

package idtools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIDMap(t *testing.T) {
	type tests struct {
		mapSpec    []string
		mapSetting string
		fail       bool
	}
	testList := []tests{
		{[]string{"0:1:100000"}, "uid", false},
		{[]string{"0:1:100000", "5:200000:100"}, "gid", false},
		{[]string{"0:1:100000", "5:200000:x100"}, "gid", true},
		{[]string{"0:1:100000"}, "uid", false},
		{[]string{"0:1:1000000000000000"}, "uid", true},
		{[]string{"b0:1:100000"}, "uid", true},
		{[]string{"0:b1:100000"}, "uid", true},
		{[]string{"0:1:1000b00"}, "uid", true},
		{[]string{"0100000"}, "uid", true},
	}

	for _, test := range testList {
		_, err := ParseIDMap(test.mapSpec, test.mapSetting)
		if test.fail {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
