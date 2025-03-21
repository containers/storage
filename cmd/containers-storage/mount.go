package main

import (
	"fmt"
	"os"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

type mountPointOrError struct {
	ID         string `json:"id"`
	MountPoint string `json:"mountpoint"`
	Error      string `json:"error"`
}
type mountPointError struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

func mount(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	moes := []mountPointOrError{}
	for _, arg := range args {
		if paramReadOnly {
			result, err := m.MountImage(arg, []string{}, paramMountLabel)
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			moes = append(moes, mountPointOrError{arg, result, errText})
		} else {
			result, err := m.Mount(arg, paramMountLabel)
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			moes = append(moes, mountPointOrError{arg, result, errText})
		}
	}
	if jsonOutput {
		if _, err := outputJSON(moes); err != nil {
			return 1, err
		}
	} else {
		for _, mountOrError := range moes {
			if mountOrError.Error != "" {
				fmt.Fprintf(os.Stderr, "%s while mounting %s\n", mountOrError.Error, mountOrError.ID)
			}
			fmt.Printf("%s\n", mountOrError.MountPoint)
		}
	}
	for _, mountOrErr := range moes {
		if mountOrErr.Error != "" {
			return 1, nil
		}
	}
	return 0, nil
}

func unmount(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	mes := []mountPointError{}
	errors := false
	for _, arg := range args {
		mounted, err := m.Unmount(arg, force)
		errText := ""
		if err != nil {
			var err1 error
			_, err1 = m.UnmountImage(arg, force)
			if err1 != nil {
				errText = err.Error()
				errors = true
			}
		} else {
			if !mounted {
				fmt.Printf("%s mountpoint unmounted\n", arg)
			}
		}
		mes = append(mes, mountPointError{arg, errText})
	}
	if jsonOutput {
		if _, err := outputJSON(mes); err != nil {
			return 1, err
		}
	} else {
		for _, me := range mes {
			if me.Error != "" {
				fmt.Fprintf(os.Stderr, "%s while unmounting %s\n", me.Error, me.ID)
			}
		}
	}
	if errors {
		return 1, nil
	}
	return 0, nil
}

func mounted(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	mes := []mountPointError{}
	errors := false
	for _, arg := range args {
		mounted, err := m.Mounted(arg)
		errText := ""
		if err != nil {
			errText = err.Error()
			errors = true
		} else {
			if mounted > 0 {
				fmt.Printf("%s mounted\n", arg)
			}
		}
		mes = append(mes, mountPointError{arg, errText})
	}
	if jsonOutput {
		if _, err := outputJSON(mes); err != nil {
			return 1, err
		}
	} else {
		for _, me := range mes {
			if me.Error != "" {
				fmt.Fprintf(os.Stderr, "%s checking if %s is mounted\n", me.Error, me.ID)
			}
		}
	}
	if errors {
		return 1, nil
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:       []string{"mount"},
		optionsHelp: "[options [...]] LayerOrContainerNameOrID",
		usage:       "Mount a layer or container",
		minArgs:     1,
		maxArgs:     -1,
		action:      mount,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&paramMountOptions, []string{"-opt", "o"}, "", "Mount Options")
			flags.StringVar(&paramMountLabel, []string{"-label", "l"}, "", "Mount Label")
			flags.BoolVar(&paramReadOnly, []string{"-read-only", "-ro", "r"}, paramReadOnly, "Mount image readonly")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"unmount", "umount"},
		optionsHelp: "LayerOrContainerNameOrID",
		usage:       "Unmount an image, layer or container",
		minArgs:     1,
		maxArgs:     -1,
		action:      unmount,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			flags.BoolVar(&force, []string{"-force", "f"}, jsonOutput, "Force the umount")
		},
	})
	commands = append(commands, command{
		names:       []string{"mounted"},
		optionsHelp: "LayerOrContainerNameOrID",
		usage:       "Check if a file system is mounted",
		minArgs:     1,
		maxArgs:     -1,
		action:      mounted,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
