package main

import (
	"fmt"
	"os"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

func gc(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	defer func() {
		if _, err := m.Shutdown(true); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
		}
	}()
	if err := m.GarbageCollect(); err != nil {
		return 1, err
	}
	if jsonOutput {
		return outputJSON(string(""))
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:   []string{"gc"},
		usage:   "Perform garbage collection",
		minArgs: 0,
		maxArgs: 0,
		action:  gc,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
