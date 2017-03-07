package main

import (
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

func version(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	version, err := m.Version()
	if err != nil {
		fmt.Fprintf(os.Stderr, "version: %v\n", err)
		return 1
	}
	if jsonOutput {
		return jsonEncodeToStdout(version)
	}

	for _, pair := range version {
		fmt.Fprintf(os.Stderr, "%s: %s\n", pair[0], pair[1])
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:   []string{"version"},
		usage:   "Return oci-storage version information",
		minArgs: 0,
		action:  version,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
