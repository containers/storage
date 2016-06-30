package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

var (
	rawMountLabel = ""
	rawName       = ""
	rawCreateRO   = false
)

func rawCreate(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	parent := ""
	if len(args) > 0 {
		parent = args[0]
	}
	layer, err := m.RawCreate(parent, rawName, rawMountLabel, !rawCreateRO)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if layer.Name != "" {
		fmt.Printf("%s\t%s\n", layer.ID, layer.Name)
	} else {
		fmt.Printf("%s\n", layer.ID)
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"rawcreate"},
		optionsHelp: "[options [...]] [parentRawLayerNameOrID]",
		usage:       "Create a new raw layer",
		maxArgs:     1,
		action:      rawCreate,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&rawMountLabel, []string{"-label", "l"}, "", "Mount Label")
			flags.StringVar(&rawName, []string{"-name", "n"}, "", "Layer name")
			flags.StringVar(&rawName, []string{"-readonly", "r"}, "", "Mark as read-only")
		},
	})
}
