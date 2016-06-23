package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/pkg/mflag"
)

var listAllPets = false

func petMatch(p Pet, id string) bool {
	return id == "" || id == p.Name() || len(id) >= 12 && strings.HasPrefix(p.ID(), id)
}

func pets(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	pets, err := m.Pets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	for _, pet := range pets {
		if petMatch(pet, name) || listAllPets {
			if pet.Name() != "" {
				fmt.Printf("%s\t%s\n", pet.ID(), pet.Name())
			} else {
				fmt.Printf("%s\n", pet.ID())
			}
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"pets"},
		usage:       "List pets",
		optionsHelp: "[options [...]]",
		maxArgs:     1,
		action:      pets,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&listAllPets, []string{"-all", "a"}, listAllPets, "List all pets")
		},
	})
}
