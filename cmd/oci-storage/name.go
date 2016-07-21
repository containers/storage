package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage/opts"
	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

func setNames(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	if err := m.SetNames(args[0], paramNames); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	names, err := m.GetNames(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(names)
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"set-names"},
		optionsHelp: "[options [...]] imageOrContainerNameOrID",
		usage:       "Set layer, image, or container name or names",
		minArgs:     1,
		action:      setNames,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.Var(opts.NewListOptsRef(&paramNames, nil), []string{"-name", "n"}, "new name")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "prefer JSON output")
		},
	})
}
