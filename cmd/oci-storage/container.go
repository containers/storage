package main

import (
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
	for _, container := range containers {
		for _, arg := range args {
			if container.ID != arg && container.Name != "" && container.Name != arg {
				continue
			}
			fmt.Printf("ID: %s\n", container.ID)
			if container.Name != "" {
				fmt.Printf("Name: %s\n", container.Name)
			}
			fmt.Printf("Image: %s\n", container.ImageID)
			for _, image := range images {
				if image.ID == container.ImageID && image.Name != "" {
					fmt.Printf("Image name: %s\n", image.Name)
				}
			}
			fmt.Printf("Layer: %s\n", container.LayerID)
		}
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
	})
}
