package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/container"
	"github.com/docker/docker/pkg/mflag"
)

var listAllContainers = false

func containerMatch(c *container.Container, id string) bool {
	return id == "" || id == c.Name || "/"+id == c.Name || len(id) >= 12 && strings.HasPrefix(c.ID, id)
}

func containers(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	driver := m.GetGraphDriver()
	containers, err := m.Containers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	for _, container := range containers {
		if containerMatch(container, name) {
			if listAllContainers || (container.Driver == driver) {
				fmt.Printf("%s\t%s\n", container.ID, container.Name[1:])
			}
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"containers"},
		optionsHelp: "[options [...]]",
		usage:       "List containers",
		maxArgs:     1,
		action:      containers,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&listAllContainers, []string{"-all", "a"}, listAllContainers, "List all containers")
		},
	})
}
