package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func commit(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "No pet read-write layer name or ID specified.\n")
		return 1
	}
	petRef := args[0]
	imageName := ""
	if len(args) > 1 {
		imageName = args[1]
	}
	id, err := m.CommitPet(petRef, imageName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if imageName != "" {
		fmt.Printf("%s\t%s\n", id, imageName)
	} else {
		fmt.Printf("%s\n", id)
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"commit"},
		optionsHelp: "petNameOrID [imageName]",
		usage:       "Create a new image from a pet read-write layer",
		minArgs:     1,
		action:      commit,
	})
}
