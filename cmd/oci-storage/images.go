package main

import (
	"fmt"
	"os"

	"github.com/containers/storage/storage"
	"github.com/containers/storage/pkg/mflag"
)

func images(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	images, err := m.Images()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	for _, image := range images {
		fmt.Printf("%s\n", image.ID)
		if image.Name != "" {
			fmt.Printf("\t%s\n", image.Name)
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"images"},
		optionsHelp: "[options [...]]",
		usage:       "List images",
		action:      images,
		maxArgs:     0,
	})
}
