package main

import (
	"fmt"
	"strings"

	"github.com/docker/docker/cow"
	"github.com/docker/docker/pkg/mflag"
)

func metadata(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	allExist := true
	for _, what := range args {
		exists := false
		if container, err := m.GetContainer(what); err == nil {
			exists = true
			fmt.Printf("%s: %s\n", what, strings.TrimSuffix(container.Metadata, "\n"))
		} else if image, err := m.GetImage(what); err == nil {
			exists = true
			fmt.Printf("%s: %s\n", what, strings.TrimSuffix(image.Metadata, "\n"))
		}
		allExist = allExist && exists
	}
	if !allExist {
		return 1
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"metadata"},
		optionsHelp: "[ImageOrContainerNameOrID [...]]",
		usage:       "Retrieve image or container metadata",
		minArgs:     1,
		action:      metadata,
	})
}
