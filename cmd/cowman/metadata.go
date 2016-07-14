package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/docker/cow"
	"github.com/docker/docker/pkg/mflag"
)

var metadataQuiet = false

func metadata(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	allExist := true
	for _, what := range args {
		exists := false
		if container, err := m.GetContainer(what); err == nil {
			exists = true
			if metadataQuiet {
				fmt.Printf("%s\n", strings.TrimSuffix(container.Metadata, "\n"))
			} else {
				fmt.Printf("%s: %s\n", what, strings.TrimSuffix(container.Metadata, "\n"))
			}
		} else if image, err := m.GetImage(what); err == nil {
			exists = true
			if metadataQuiet {
				fmt.Printf("%s\n", strings.TrimSuffix(image.Metadata, "\n"))
			} else {
				fmt.Printf("%s: %s\n", what, strings.TrimSuffix(image.Metadata, "\n"))
			}
		}
		allExist = allExist && exists
	}
	if !allExist {
		return 1
	}
	return 0
}

func setMetadata(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	if MetadataFile == "" && Metadata == "" {
		fmt.Fprintf(os.Stderr, "no new metadata provided\n")
		return 1
	}
	if MetadataFile != "" {
		f, err := os.Open(MetadataFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		b, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		Metadata = string(b)
	}
	if err := m.SetMetadata(args[0], Metadata); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
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
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&metadataQuiet, []string{"-quiet", "q"}, metadataQuiet, "Don't print names and IDs")
		},
	})
	commands = append(commands, command{
		names:       []string{"set-metadata"},
		optionsHelp: "[options [...]] imageOrContainerNameOrID",
		usage:       "Set image or container metadata",
		minArgs:     1,
		maxArgs:     1,
		action:      setMetadata,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&Metadata, []string{"-metadata", "m"}, "", "Metadata")
			flags.StringVar(&MetadataFile, []string{"-metadata-file", "f"}, "", "Metadata File")
		},
	})
}
