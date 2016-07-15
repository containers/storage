package main

import (
	"fmt"

	"github.com/containers/storage/storage"
	"github.com/containers/storage/pkg/mflag"
)

var existQuiet = false

func exist(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	allExist := true
	for _, what := range args {
		exists := m.Exists(what)
		if !existQuiet {
			fmt.Printf("%s: %v\n", what, exists)
		}
		allExist = allExist && exists
	}
	if !allExist {
		return 1
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"exists"},
		optionsHelp: "[LayerOrImageOrContainerNameOrID [...]]",
		usage:       "Check if a layer or image or container exists",
		minArgs:     1,
		action:      exist,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&existQuiet, []string{"-quiet", "q"}, existQuiet, "Don't print names")
		},
	})
}
