package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

func wipe(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	err := m.Wipe()
	if jsonOutput {
		if err == nil {
			json.NewEncoder(os.Stdout).Encode(string(""))
		} else {
			json.NewEncoder(os.Stdout).Encode(err)
			return 1, nil
		}
	} else {
		if err != nil {
			return 1, fmt.Errorf("%s: %+v", action, err)
		}
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:   []string{"wipe"},
		usage:   "Wipe all layers, images, and containers",
		minArgs: 0,
		action:  wipe,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
