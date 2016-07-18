package main

import (
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

func deleteThing(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
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
		usage:       "Delete a layer or image or container",
		minArgs:     1,
		action:      deleteThing,
	})
}
