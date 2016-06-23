package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

var loadVerbose = false

func load(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) > 0 {
		for _, arg := range args {
			f, err := os.Open(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				return 1
			}
			m.Load(os.Stdout, !loadVerbose, f)
			f.Close()
		}
	} else {
		m.Load(os.Stdout, !loadVerbose, os.Stdin)
	}
	return 0
}

func save(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	m.Save(os.Stdout, args)
	return 0
}

func init() {
	commands = append(commands, command{
		names:  []string{"load"},
		usage:  "Load images",
		action: load,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&loadVerbose, []string{"-verbose"}, loadVerbose, "Verbose progress")
		},
	})
	commands = append(commands, command{
		names:       []string{"save"},
		optionsHelp: "imageID [...]",
		usage:       "Save images",
		action:      save,
	})
}
