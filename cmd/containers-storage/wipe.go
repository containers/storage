package main

import (
	"fmt"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

func wipe(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	err := m.Wipe()
	if err != nil {
		if jsonOutput {
			_, err2 := outputJSON(err)
			return 1, err2 // Note that err2 is usually nil
		}
		return 1, fmt.Errorf("%s: %+v", action, err)
	}
	if jsonOutput {
		return outputJSON(string(""))
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:   []string{"wipe"},
		usage:   "Wipe all layers, images, and containers",
		minArgs: 0,
		action:  wipe,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
