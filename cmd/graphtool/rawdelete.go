package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func rawDelete(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	for _, what := range args {
		err := m.RawDelete(what)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
			return 1
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"rawdelete"},
		optionsHelp: "[rawLayerNameOrID [...]]",
		usage:       "Delete a raw layer",
		minArgs:     1,
		action:      rawDelete,
	})
}
