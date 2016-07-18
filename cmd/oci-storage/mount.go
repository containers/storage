package main

import (
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

func mount(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	for _, arg := range args {
		result, err := m.Mount(arg, paramMountLabel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s while mounting %s\n", err, arg)
			return 1
		}
		fmt.Printf("%s\n", result)
	}
	return 0
}

func unmount(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
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
		optionsHelp: "[options [...]] LayerOrContainerNameOrID",
		usage:       "Mount a layer or container",
		minArgs:     1,
		action:      mount,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&paramMountLabel, []string{"-label", "l"}, "", "Mount Label")
		},
	})
	commands = append(commands, command{
		names:       []string{"unmount", "umount"},
		optionsHelp: "LayerOrContainerNameOrID",
		usage:       "Unmount a layer or container",
		minArgs:     1,
		action:      unmount,
	})
}
