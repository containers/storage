package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/cow"
	"github.com/docker/docker/pkg/mflag"
)

func deleteLayer(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	for _, what := range args {
		err := m.Delete(what)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
			return 1
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"delete"},
		optionsHelp: "[LayerNameOrID [...]]",
		usage:       "Delete a layer",
		minArgs:     1,
		action:      deleteLayer,
	})
}
