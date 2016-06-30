package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func rawMount(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	for _, arg := range args {
		result, err := m.RawMount(arg, rawMountLabel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s while mounting %s\n", err, arg)
			return 1
		}
		fmt.Printf("%s\n", result)
	}
	return 0
}

func rawUnmount(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	for _, arg := range args {
		if err := m.RawUnmount(arg); err != nil {
			fmt.Fprintf(os.Stderr, "%s while unmounting %s\n", err, arg)
			return 1
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"rawmount"},
		optionsHelp: "[options [...]] rawLayerNameOrID",
		usage:       "Mount a raw layer",
		minArgs:     1,
		action:      rawMount,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&rawMountLabel, []string{"-label", "l"}, "", "Mount Label")
		},
	})
	commands = append(commands, command{
		names:       []string{"rawunmount", "rawumount"},
		optionsHelp: "rawLayerNameOrID",
		usage:       "Unmount a raw layer",
		minArgs:     1,
		action:      rawUnmount,
	})
}
