package main

import (
	"fmt"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

var forceShutdown = false

func shutdown(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	_, err := m.Shutdown(forceShutdown)
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
		names:   []string{"shutdown"},
		usage:   "Shut down layer storage",
		minArgs: 0,
		maxArgs: 0,
		action:  shutdown,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			flags.BoolVar(&forceShutdown, []string{"-force", "f"}, forceShutdown, "Unmount mounted layers first")
		},
	})
}
