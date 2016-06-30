package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/mflag"
)

var listRawLayersTree = false

func rawlayers(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	layers, err := m.RawLayers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	nodes := []TreeNode{}
	for _, layer := range layers {
		if listRawLayersTree {
			node := TreeNode{
				left:  string(layer.Parent),
				right: string(layer.ID),
				notes: []string{},
			}
			if node.left == "" {
				node.left = "(base)"
			}
			if layer.Name != "" {
				node.notes = append(node.notes, layer.Name)
			}
			if layer.MountPoint != "" {
				node.notes = append(node.notes, layer.MountPoint)
			}
			nodes = append(nodes, node)
		} else {
			fmt.Printf("%s\n", layer.ID)
			if layer.Name != "" {
				fmt.Printf("\t%s\n", layer.Name)
			}
		}
	}
	if listRawLayersTree {
		PrintTree(nodes)
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"rawlayers"},
		optionsHelp: "[options [...]]",
		usage:       "List raw layers",
		action:      rawlayers,
		maxArgs:     0,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&listRawLayersTree, []string{"-tree", "t"}, listRawLayersTree, "Use a tree")
		},
	})
}
