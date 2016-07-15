package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/cow"
	"github.com/docker/docker/pkg/mflag"
)

func containers(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	containers, err := m.Containers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	for _, container := range containers {
		fmt.Printf("%s\n", container.ID)
		if container.Name != "" {
			fmt.Printf("\t%s\n", container.Name)
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"containers"},
		optionsHelp: "[options [...]]",
		usage:       "List containers",
		action:      containers,
		maxArgs:     0,
	})
}
