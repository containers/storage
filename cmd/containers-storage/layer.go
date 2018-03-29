package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

func layer(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	matched := []*storage.Layer{}
	for _, arg := range args {
		if layer, err := m.Layer(arg); err == nil {
			matched = append(matched, layer)
		}
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(matched)
	} else {
		for _, layer := range matched {
			fmt.Printf("ID: %s\n", layer.ID)
			if layer.Parent != "" {
				fmt.Printf("Parent: %s\n", layer.Parent)
			}
			for _, name := range layer.Names {
				fmt.Printf("Name: %s\n", name)
			}
		}
	}
	if len(matched) != len(args) {
		return 1
	}
	return 0
}

func layerParentOwners(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	matched := []*storage.Layer{}
	for _, arg := range args {
		if layer, err := m.Layer(arg); err == nil {
			matched = append(matched, layer)
		}
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(matched)
	} else {
		for _, layer := range matched {
			uids, gids, err := m.LayerParentOwners(layer.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "LayerParentOwner: %v\n", err)
				return 1
			}
			fmt.Printf("ID: %s\n", layer.ID)
			if len(uids) > 0 {
				fmt.Printf("UIDs: %v\n", uids)
			}
			if len(gids) > 0 {
				fmt.Printf("GIDs: %v\n", gids)
			}
		}
	}
	if len(matched) != len(args) {
		return 1
	}
	return 0
}

func init() {
	commands = append(commands,
		command{
			names:       []string{"layer"},
			optionsHelp: "[options [...]] layerNameOrID [...]",
			usage:       "Examine a layer",
			action:      layer,
			minArgs:     1,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			},
		},
		command{
			names:       []string{"layer-parent-owners"},
			optionsHelp: "[options [...]] layerNameOrID [...]",
			usage:       "Compute the set of unmapped parent UIDs and GIDs of the layer",
			action:      layerParentOwners,
			minArgs:     1,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			},
		},
	)
}
