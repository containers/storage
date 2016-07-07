package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/cow"
	"github.com/docker/docker/pkg/mflag"
)

var listLayersTree = false

func layers(flags *mflag.FlagSet, action string, m cow.Mall, args []string) int {
	layers, err := m.Layers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	nodes := []TreeNode{}
	for _, layer := range layers {
		if listLayersTree {
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
	if listLayersTree {
		PrintTree(nodes)
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"layers"},
		optionsHelp: "[options [...]]",
		usage:       "List layers",
		action:      layers,
		maxArgs:     0,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&listLayersTree, []string{"-tree", "t"}, listLayersTree, "Use a tree")
		},
	})
}
