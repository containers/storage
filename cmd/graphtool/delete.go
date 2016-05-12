package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func deletepet(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	var name string

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "No pet filesystem specified.\n")
		return 1
	}
	name = args[0]
	pstore, err := m.GetPetStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	p, err := pstore.Get(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting pet %q: %v\n", name, err)
		return 1
	}
	err = pstore.Remove(p.ID(), p.Name())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error releasing pet layer %q: %v\n", name, err)
		return 1
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"delete", "release"},
		optionsHelp: "layerName",
		usage:       "Delete (release) a pet read-write layer",
		minArgs:     1,
		action:      deletepet,
	})
}
