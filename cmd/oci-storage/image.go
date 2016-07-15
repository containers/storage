package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/cow"
	"github.com/docker/docker/pkg/mflag"
)

func image(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	images, err := m.Images()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	for _, image := range images {
		for _, arg := range args {
			if image.ID != arg && image.Name != "" && image.Name != arg {
				continue
			}
			fmt.Printf("ID: %s\n", image.ID)
			if image.Name != "" {
				fmt.Printf("Name: %s\n", image.Name)
			}
			fmt.Printf("Top Layer: %s\n", image.TopLayer)
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"image"},
		optionsHelp: "[options [...]] imageNameOrID [...]",
		usage:       "Examine an image",
		action:      image,
		minArgs:     1,
	})
}
