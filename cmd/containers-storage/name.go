package main

import (
	"fmt"

	"github.com/containers/storage"
	"github.com/containers/storage/internal/opts"
	"github.com/containers/storage/pkg/mflag"
)

func getNames(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
	}
	id, err := m.Lookup(args[0])
	if err != nil {
		return 1, err
	}
	names, err := m.Names(id)
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		return outputJSON(append([]string{}, names...))
	}
	for _, name := range names {
		fmt.Printf("%s\n", name)
	}
	return 0, nil
}

func addNames(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
	}
	id, err := m.Lookup(args[0])
	if err != nil {
		return 1, err
	}
	if err := m.AddNames(id, paramNames); err != nil {
		return 1, err
	}
	names, err := m.Names(id)
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		if _, err := outputJSON(names); err != nil {
			return 1, err
		}
	}
	return 0, nil
}

func removeNames(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
	}
	id, err := m.Lookup(args[0])
	if err != nil {
		return 1, err
	}
	if err := m.RemoveNames(id, paramNames); err != nil {
		return 1, err
	}
	names, err := m.Names(id)
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		if _, err := outputJSON(names); err != nil {
			return 1, err
		}
	}
	return 0, nil
}

func setNames(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
	}
	id, err := m.Lookup(args[0])
	if err != nil {
		return 1, err
	}
	if err := m.SetNames(id, paramNames); err != nil {
		return 1, err
	}
	names, err := m.Names(id)
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		if _, err := outputJSON(names); err != nil {
			return 1, err
		}
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:       []string{"get-names", "getnames"},
		optionsHelp: "[options [...]] imageOrContainerNameOrID",
		usage:       "Get layer, image, or container name or names",
		minArgs:     1,
		maxArgs:     1,
		action:      getNames,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"add-names", "addnames"},
		optionsHelp: "[options [...]] imageOrContainerNameOrID",
		usage:       "Add layer, image, or container name or names",
		minArgs:     1,
		maxArgs:     -1,
		action:      addNames,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.Var(opts.NewListOptsRef(&paramNames, nil), []string{"-name", "n"}, "Name to add")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"remove-names", "removenames"},
		optionsHelp: "[options [...]] imageOrContainerNameOrID",
		usage:       "Remove layer, image, or container name or names",
		minArgs:     1,
		maxArgs:     -1,
		action:      removeNames,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.Var(opts.NewListOptsRef(&paramNames, nil), []string{"-name", "n"}, "Name to remove")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"set-names", "setnames"},
		optionsHelp: "[options [...]] imageOrContainerNameOrID",
		usage:       "Set layer, image, or container name or names",
		minArgs:     1,
		maxArgs:     -1,
		action:      setNames,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.Var(opts.NewListOptsRef(&paramNames, nil), []string{"-name", "n"}, "New name")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
