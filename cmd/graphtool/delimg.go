package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

func deleteimage(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "No image name or reference specified.\n")
		return 1
	}
	if err := m.DeleteImage(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"delete-image"},
		optionsHelp: "imageRef",
		usage:       "Delete an image",
		minArgs:     1,
		action:      deleteimage,
	})
}
