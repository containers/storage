package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

var existQuiet = false

func exist(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	anyMissing := false
	existDict := make(map[string]bool)
	for _, what := range args {
		exists := m.Exists(what)
		existDict[what] = exists
		if !exists {
			anyMissing = true
		}
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(existDict)
	} else {
		if !existQuiet {
			for what, exists := range existDict {
				fmt.Printf("%s: %v\n", what, exists)
			}
		}
	}
	if anyMissing {
		return 1
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"exists"},
		optionsHelp: "[LayerOrImageOrContainerNameOrID [...]]",
		usage:       "Check if a layer or image or container exists",
		minArgs:     1,
		action:      exist,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&existQuiet, []string{"-quiet", "q"}, existQuiet, "Don't print names")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
