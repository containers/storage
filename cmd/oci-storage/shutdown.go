package main

import (
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

var (
	forceShutdown = false
)

func shutdown(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	_, err := m.Shutdown(forceShutdown)
	if jsonOutput {
		if err != nil {
			return jsonEncodeToStdout(err)
		}
		return jsonEncodeToStdout("")
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", action, err)
		return 1
	}

	return 0
}

func init() {
	commands = append(commands, command{
		names:   []string{"shutdown"},
		usage:   "Shut down layer storage",
		minArgs: 0,
		action:  shutdown,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			flags.BoolVar(&forceShutdown, []string{"-force", "f"}, forceShutdown, "Unmount mounted layers first")
		},
	})
}
