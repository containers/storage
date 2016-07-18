package main

import (
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

var listLayersTree = false

func layers(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	layers, err := m.Layers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	imageMap := make(map[string]storage.Image)
	if images, err := m.Images(); err == nil {
		for _, image := range images {
			imageMap[image.TopLayer] = image
		}
	}
	containerMap := make(map[string]storage.Container)
	if containers, err := m.Containers(); err == nil {
		for _, container := range containers {
			containerMap[container.LayerID] = container
		}
	}
	nodes := []treeNode{}
	for _, layer := range layers {
		if listLayersTree {
			node := treeNode{
				left:  string(layer.Parent),
				right: string(layer.ID),
				notes: []string{},
			}
			if node.left == "" {
				node.left = "(base)"
			}
			if layer.Name != "" {
				node.notes = append(node.notes, "name: "+layer.Name)
			}
			if layer.MountPoint != "" {
				node.notes = append(node.notes, "mount: "+layer.MountPoint)
			}
			if image, ok := imageMap[layer.ID]; ok {
				node.notes = append(node.notes, fmt.Sprintf("image: %s", image.ID))
				if image.Name != "" {
					node.notes = append(node.notes, fmt.Sprintf("image name: %s", image.Name))
				}
			}
			if container, ok := containerMap[layer.ID]; ok {
				node.notes = append(node.notes, fmt.Sprintf("container: %s", container.ID))
				if container.Name != "" {
					node.notes = append(node.notes, fmt.Sprintf("container name: %s", container.Name))
				}
			}
			nodes = append(nodes, node)
		} else {
			fmt.Printf("%s\n", layer.ID)
			if layer.Name != "" {
				fmt.Printf("\tname: %s\n", layer.Name)
			}
			if image, ok := imageMap[layer.ID]; ok {
				fmt.Printf("\timage: %s\n", image.ID)
				if image.Name != "" {
					fmt.Printf("\t\tname: %s\n", image.Name)
				}
			}
			if container, ok := containerMap[layer.ID]; ok {
				fmt.Printf("\tcontainer: %s\n", container.ID)
				if container.Name != "" {
					fmt.Printf("\t\tname: %s\n", container.Name)
				}
			}
		}
	}
	if listLayersTree {
		printTree(nodes)
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
