package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

var mountLabel = ""

func create(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "No image name or ID specified.\n")
		return 1
	}
	imageRef := args[0]
	petName := ""
	if len(args) > 1 {
		petName = args[1]
	}
	id, err := m.CreatePet(imageRef, petName, mountLabel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if petName != "" {
		fmt.Printf("%s\t%s\n", id, petName)
	} else {
		fmt.Printf("%s\n", id)
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"create"},
		optionsHelp: "imageID [name]",
		usage:       "Create a new pet read-write layer from an image",
		minArgs:     1,
		action:      create,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&mountLabel, []string{"-label", "l"}, "", "Mount Label")
		},
	})
}
