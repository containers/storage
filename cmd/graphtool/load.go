package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func load(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	e, err := m.GetImageExporter(logImageEvent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	if len(args) > 0 {
		for _, arg := range args {
			f, err := os.Open(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			e.Load(f, os.Stdout, true)
			f.Close()
		}
	} else {
		e.Load(os.Stdin, os.Stdout, true)
	}
	return 0
}

func save(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	e, err := m.GetImageExporter(logImageEvent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	e.Save(args, os.Stdout)
	return 0
}

func init() {
	commands = append(commands, command{
		names:  []string{"load"},
		usage:  "Load images",
		action: load,
	})
	commands = append(commands, command{
		names:       []string{"save"},
		optionsHelp: "imageID [...]",
		usage:       "Save images",
		action:      save,
	})
}
