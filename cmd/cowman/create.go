package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/cow"
	"github.com/docker/docker/pkg/mflag"
)

var (
	MountLabel = ""
	Name       = ""
	CreateRO   = false
)

func create(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	parent := ""
	if len(args) > 0 {
		parent = args[0]
	}
	layer, err := m.Create(parent, Name, MountLabel, !CreateRO)
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
		names:       []string{"create"},
		optionsHelp: "[options [...]] [parentLayerNameOrID]",
		usage:       "Create a new layer",
		maxArgs:     1,
		action:      create,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&MountLabel, []string{"-label", "l"}, "", "Mount Label")
			flags.StringVar(&Name, []string{"-name", "n"}, "", "Layer name")
			flags.BoolVar(&CreateRO, []string{"-readonly", "r"}, false, "Mark as read-only")
		},
	})
}
