package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func rawExists(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	allExist := true
	for _, what := range args {
		exists := m.RawExists(what)
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
		names:       []string{"rawexists"},
		optionsHelp: "[rawLayerNameOrID [...]]",
		usage:       "Check if a raw layer exists",
		minArgs:     1,
		action:      rawExists,
	})
}
