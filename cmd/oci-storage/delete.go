package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

func deleteThing(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	deleted := make(map[string]error)
	for _, what := range args {
		err := m.Delete(what)
		deleted[what] = err
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(deleted)
	} else {
		for what, err := range deleted {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
			}
		}
	}
	for _, err := range deleted {
		if err != nil {
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
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
