package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/cow"
	"github.com/docker/docker/pkg/mflag"
)

func exist(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	allExist := true
	for _, what := range args {
		exists := m.Exists(what)
		fmt.Fprintf(os.Stderr, "%s: %v\n", what, exists)
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
		optionsHelp: "[LayerNameOrID [...]]",
		usage:       "Check if a layer exists",
		minArgs:     1,
		action:      exist,
	})
}
