package cri

import (
	"strings"
)

func matchesNameOrID(filter, ID string, names []string) bool {
	if filter == ID {
		return true
	}
	for _, name := range names {
		if filter == name {
			return true
		}
	}
	if len(filter) < 7 {
		return false
	}
	return strings.EqualFold(filter, ID[:len(filter)])
}
