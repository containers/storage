package main

import (
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

func status(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	status, err := m.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}
	if jsonOutput {
		return jsonEncodeToStdout(status)
	}

	for _, pair := range status {
		fmt.Fprintf(os.Stderr, "%s: %s\n", pair[0], pair[1])
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:   []string{"status"},
		usage:   "Check on graph driver status",
		minArgs: 0,
		action:  status,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
