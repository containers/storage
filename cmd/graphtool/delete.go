package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func deletepet(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	var name string

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "No read-write layer specified.\n")
		return 1
	}
	name = args[0]
	lstore, err := m.GetLayerStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	l, err := lstore.GetRWLayer(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting layer %q: %v\n", name, err)
		return 1
	}
	_, err = lstore.ReleaseRWLayer(l)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error releasing layer %q: %v\n", name, err)
		return 1
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"delete", "release"},
		optionsHelp: "layerName",
		usage:       "Delete (release) a read-write layer",
		minArgs:     1,
		action:      deletepet,
	})
}
