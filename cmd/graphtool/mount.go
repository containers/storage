package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func mount(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	for _, arg := range args {
		result, err := m.Mount(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s while mounting %s\n", err, arg)
			return 1
		}
		fmt.Printf("%s\n", result)
	}
	return 0
}

func unmount(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	for _, arg := range args {
		if err := m.Unmount(arg); err != nil {
			fmt.Fprintf(os.Stderr, "%s while unmounting %s\n", err, arg)
			return 1
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"mount"},
		optionsHelp: "petNameOrContainerName",
		usage:       "Mount a read-write layer",
		minArgs:     1,
		action:      mount,
	})
	commands = append(commands, command{
		names:       []string{"unmount", "umount"},
		optionsHelp: "petNameOrContainerName",
		usage:       "Unmount a read-write layer",
		minArgs:     1,
		action:      unmount,
	})
}
