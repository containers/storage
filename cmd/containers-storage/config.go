package main

import (
	"fmt"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/types"
)

func config(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	options, err := types.DefaultStoreOptions()
	if err != nil {
		return 1, fmt.Errorf("default: %+v", err)
	}
	if len(args) > 0 {
		if err = types.ReloadConfigurationFileIfNeeded(args[0], &options); err != nil {
			return 1, fmt.Errorf("reload: %+v", err)
		}
	}
	return outputJSON(options)
}

func init() {
	commands = append(commands, command{
		names:       []string{"config"},
		usage:       "Print storage library configuration as JSON",
		minArgs:     0,
		maxArgs:     1,
		optionsHelp: "[configurationFile]",
		action:      config,
	})
}
