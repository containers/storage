package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func rawWipe(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	err := m.RawWipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", err)
		return 1
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:   []string{"rawwipe"},
		usage:   "Wipe all raw layers",
		minArgs: 0,
		action:  rawWipe,
	})
}
