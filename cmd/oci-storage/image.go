package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

func image(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	images, err := m.Images()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	matched := []storage.Image{}
	for _, image := range images {
		for _, arg := range args {
			if image.ID != arg && image.Name != "" && image.Name != arg {
				continue
			}
			matched = append(matched, image)
		}
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(matched)
	} else {
		for _, image := range matched {
			fmt.Printf("ID: %s\n", image.ID)
			if image.Name != "" {
				fmt.Printf("Name: %s\n", image.Name)
			}
			fmt.Printf("Top Layer: %s\n", image.TopLayer)
		}
	}
	if len(matched) != len(args) {
		return 1
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
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "prefer JSON output")
		},
	})
}
