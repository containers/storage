package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

type mountPointOrError struct {
	ID         string `json:"id"`
	MountPoint string `json:"mountpoint"`
	Error      error  `json:"error"`
}
type mountPointError struct {
	ID    string `json:"id"`
	Error error  `json:"error"`
}

func mount(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	moes := []mountPointOrError{}
	for _, arg := range args {
		result, err := m.Mount(arg, paramMountLabel)
		moes = append(moes, mountPointOrError{arg, result, err})
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(moes)
	} else {
		for _, mountOrError := range moes {
			if mountOrError.Error != nil {
				fmt.Fprintf(os.Stderr, "%s while mounting %s\n", mountOrError.Error, mountOrError.ID)
			}
			fmt.Printf("%s\n", mountOrError.MountPoint)
		}
	}
	for _, mountOrErr := range moes {
		if mountOrErr.Error != nil {
			return 1
		}
	}
	return 0
}

func unmount(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	mes := []mountPointError{}
	errors := false
	for _, arg := range args {
		err := m.Unmount(arg)
		if err != nil {
			errors = true
		}
		mes = append(mes, mountPointError{arg, err})
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(mes)
	} else {
		for _, me := range mes {
			if me.Error != nil {
				fmt.Fprintf(os.Stderr, "%s while unmounting %s\n", me.Error, me.ID)
			}
		}
	}
	if errors {
		return 1
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
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"unmount", "umount"},
		optionsHelp: "LayerOrContainerNameOrID",
		usage:       "Unmount a layer or container",
		minArgs:     1,
		action:      unmount,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "prefer JSON output")
		},
	})
}
