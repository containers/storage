package main

import (
	"fmt"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

func containers(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	containers, err := m.Containers()
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		return outputJSON(containers)
	}
	for _, container := range containers {
		fmt.Printf("%s\n", container.ID)
		for _, name := range container.Names {
			fmt.Printf("\tname: %s\n", name)
		}
		for _, name := range container.BigDataNames {
			fmt.Printf("\tdata: %s\n", name)
		}
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:       []string{"containers"},
		optionsHelp: "[options [...]]",
		usage:       "List containers",
		action:      containers,
		maxArgs:     0,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
