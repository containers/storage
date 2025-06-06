package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

var (
	listLayersTree  = false
	listLayersQuick = false
)

func storageLayers(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	driver, err := m.GraphDriver()
	if err != nil {
		return 1, err
	}
	layers, err := driver.ListLayers()
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(layers); err != nil {
			return 1, err
		}
		return 0, nil
	}
	for _, layer := range layers {
		fmt.Printf("%s\n", layer)
	}
	return 0, nil
}

func layers(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	layers, err := m.Layers()
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		return outputJSON(layers)
	}
	imageMap := make(map[string]*[]storage.Image)
	if images, err := m.Images(); err == nil {
		for _, image := range images {
			for _, layerID := range append([]string{image.TopLayer}, image.MappedTopLayers...) {
				if ilist, ok := imageMap[layerID]; ok && ilist != nil {
					list := append(*ilist, image)
					imageMap[layerID] = &list
				} else {
					list := []storage.Image{image}
					imageMap[layerID] = &list
				}
			}
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
				left:  layer.Parent,
				right: layer.ID,
				notes: []string{},
			}
			if node.left == "" {
				node.left = "(base)"
			}
			for _, name := range layer.Names {
				node.notes = append(node.notes, "name: "+name)
			}
			if layer.MountPoint != "" {
				node.notes = append(node.notes, "mount: "+layer.MountPoint)
			}
			if imageList, ok := imageMap[layer.ID]; ok && imageList != nil {
				for _, image := range *imageList {
					node.notes = append(node.notes, fmt.Sprintf("image: %s", image.ID))
					for _, name := range image.Names {
						node.notes = append(node.notes, fmt.Sprintf("image name: %s", name))
					}
				}
			}
			if container, ok := containerMap[layer.ID]; ok {
				node.notes = append(node.notes, fmt.Sprintf("container: %s", container.ID))
				for _, name := range container.Names {
					node.notes = append(node.notes, fmt.Sprintf("container name: %s", name))
				}
			}
			nodes = append(nodes, node)
		} else {
			fmt.Printf("%s\n", layer.ID)
			if listLayersQuick {
				continue
			}
			for _, name := range layer.Names {
				fmt.Printf("\tname: %s\n", name)
			}
			if imageList, ok := imageMap[layer.ID]; ok && imageList != nil {
				for _, image := range *imageList {
					fmt.Printf("\timage: %s\n", image.ID)
					for _, name := range image.Names {
						fmt.Printf("\t\tname: %s\n", name)
					}
				}
			}
			if container, ok := containerMap[layer.ID]; ok {
				fmt.Printf("\tcontainer: %s\n", container.ID)
				for _, name := range container.Names {
					fmt.Printf("\t\tname: %s\n", name)
				}
			}
		}
	}
	if listLayersTree {
		printTree(nodes)
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:       []string{"layers"},
		optionsHelp: "[options [...]]",
		usage:       "List layers",
		action:      layers,
		minArgs:     0,
		maxArgs:     0,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&listLayersTree, []string{"-tree", "t"}, listLayersTree, "Use a tree")
			flags.BoolVar(&listLayersQuick, []string{"-quick", "q"}, listLayersTree, "Just the IDs")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"storage-layers"},
		optionsHelp: "[options [...]]",
		usage:       "List storage layers",
		action:      storageLayers,
		minArgs:     0,
		maxArgs:     0,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
