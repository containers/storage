package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

func container(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	images, err := m.Images()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	containers, err := m.Containers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	matches := []storage.Container{}
	for _, container := range containers {
	nextContainer:
		for _, arg := range args {
			if container.ID == arg {
				matches = append(matches, container)
				break nextContainer
			}
			for _, name := range container.Names {
				if name == arg {
					matches = append(matches, container)
					break nextContainer
				}
			}
		}
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(matches)
	} else {
		for _, container := range matches {
			fmt.Printf("ID: %s\n", container.ID)
			for _, name := range container.Names {
				fmt.Printf("Name: %s\n", name)
			}
			fmt.Printf("Image: %s\n", container.ImageID)
			for _, image := range images {
				if image.ID == container.ImageID {
					for _, name := range image.Names {
						fmt.Printf("Image name: %s\n", name)
					}
					break
				}
			}
			fmt.Printf("Layer: %s\n", container.LayerID)
		}
	}
	if len(matches) != len(args) {
		return 1
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"container"},
		optionsHelp: "[options [...]] containerNameOrID [...]",
		usage:       "Examine a container",
		action:      container,
		minArgs:     1,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "prefer JSON output")
		},
	})
}
