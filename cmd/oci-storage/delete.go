package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

var testDeleteImage = false

func deleteThing(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	deleted := make(map[string]error)
	for _, what := range args {
		err := m.Delete(what)
		deleted[what] = err
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(deleted)
	} else {
		for what, err := range deleted {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
			}
		}
	}
	for _, err := range deleted {
		if err != nil {
			return 1
		}
	}
	return 0
}

func deleteLayer(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	deleted := make(map[string]error)
	for _, what := range args {
		err := m.DeleteLayer(what)
		deleted[what] = err
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(deleted)
	} else {
		for what, err := range deleted {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
			}
		}
	}
	for _, err := range deleted {
		if err != nil {
			return 1
		}
	}
	return 0
}

type deletedImage struct {
	DeletedLayers []string `json:"deleted-layers,omitifempty"`
	Error         error    `json:"error,omitifempty"`
}

func deleteImage(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	deleted := make(map[string]deletedImage)
	for _, what := range args {
		layers, err := m.DeleteImage(what, !testDeleteImage)
		deleted[what] = deletedImage{layers, err}
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(deleted)
	} else {
		for what, record := range deleted {
			if record.Error != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", what, record.Error)
			} else {
				for _, layer := range record.DeletedLayers {
					fmt.Fprintf(os.Stderr, "%s: %s\n", what, layer)
				}
			}
		}
	}
	for _, record := range deleted {
		if record.Error != nil {
			return 1
		}
	}
	return 0
}

func deleteContainer(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	deleted := make(map[string]error)
	for _, what := range args {
		err := m.DeleteContainer(what)
		deleted[what] = err
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(deleted)
	} else {
		for what, err := range deleted {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
			}
		}
	}
	for _, err := range deleted {
		if err != nil {
			return 1
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"delete"},
		optionsHelp: "[LayerOrImageOrContainerNameOrID [...]]",
		usage:       "Delete a layer or image or container",
		minArgs:     1,
		action:      deleteThing,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"delete-layer"},
		optionsHelp: "[LayerNameOrID [...]]",
		usage:       "Delete a layer, with safety checks",
		minArgs:     1,
		action:      deleteLayer,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"delete-image"},
		optionsHelp: "[ImageNameOrID [...]]",
		usage:       "Delete an image, with safety checks",
		minArgs:     1,
		action:      deleteImage,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&testDeleteImage, []string{"-test", "t"}, jsonOutput, "Only test removal")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"delete-container"},
		optionsHelp: "[ContainerNameOrID [...]]",
		usage:       "Delete a container, with safety checks",
		minArgs:     1,
		action:      deleteContainer,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
