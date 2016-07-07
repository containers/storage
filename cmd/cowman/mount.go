package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/cow"
	"github.com/docker/docker/pkg/mflag"
)

func mount(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	for _, arg := range args {
		result, err := m.Mount(arg, MountLabel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s while mounting %s\n", err, arg)
			return 1
		}
		fmt.Printf("%s\n", result)
	}
	return 0
}

func unmount(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
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
		optionsHelp: "[options [...]] LayerNameOrID",
		usage:       "Mount a layer",
		minArgs:     1,
		action:      mount,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&MountLabel, []string{"-label", "l"}, "", "Mount Label")
		},
	})
	commands = append(commands, command{
		names:       []string{"unmount", "umount"},
		optionsHelp: "LayerNameOrID",
		usage:       "Unmount a layer",
		minArgs:     1,
		action:      unmount,
	})
}
