package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

func listDeps(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	users, err := m.Crawl(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(users)
	} else {
		if listLayersTree {
			layer := func(l *storage.Layer) *treeNode {
				images, _ := m.GetImagesByTopLayer(l.ID)
				for _, image := range images {
					if image.TopLayer == l.ID {
						return nil
					}
				}
				container, _ := m.GetContainerByLayer(l.ID)
				if container != nil {
					return nil
				}
				node := &treeNode{left: l.Parent, right: l.ID}
				node.notes = []string{"type: layer"}
				for _, name := range l.Names {
					note := fmt.Sprintf("layer name: %s", name)
					node.notes = append(node.notes, note)
				}
				return node
			}
			image := func(i *storage.Image) treeNode {
				node := treeNode{left: i.TopLayer, right: i.ID}
				if topLayer, _ := m.GetLayer(i.TopLayer); topLayer != nil {
					node.left = topLayer.Parent
				}
				node.notes = []string{"type: image"}
				for _, name := range i.Names {
					note := fmt.Sprintf("image name: %s", name)
					node.notes = append(node.notes, note)
				}
				note := fmt.Sprintf("image top layer: %s", i.TopLayer)
				node.notes = append(node.notes, note)
				return node
			}
			container := func(c *storage.Container) treeNode {
				parent := ""
				if image, err := m.GetImage(c.ImageID); image != nil && err == nil {
					parent = image.ID
				}
				node := treeNode{left: parent, right: c.ID}
				node.notes = []string{"type: container"}
				for _, name := range c.Names {
					note := fmt.Sprintf("container name: %s", name)
					node.notes = append(node.notes, note)
				}
				note := fmt.Sprintf("container layer: %s", c.LayerID)
				node.notes = append(node.notes, note)
				return node
			}
			tree := []treeNode{}
			if c, err := m.GetContainer(users.ID); c != nil && err == nil {
				root := container(c)
				tree = append(tree, root)
			} else if i, err := m.GetImage(users.ID); i != nil && err == nil {
				root := image(i)
				tree = append(tree, root)
			} else if l, err := m.GetLayer(users.ID); l != nil && err == nil {
				root := layer(l)
				if root != nil {
					tree = append(tree, *root)
				}
			}
			for _, direct := range users.LayersDirect {
				if l, err := m.GetLayer(direct); l != nil && err == nil {
					node := layer(l)
					if node != nil {
						tree = append(tree, *node)
					}
				}
			}
			for _, indirect := range users.LayersIndirect {
				if l, err := m.GetLayer(indirect); l != nil && err == nil {
					node := layer(l)
					if node != nil {
						tree = append(tree, *node)
					}
				}
			}
			for _, direct := range users.ImagesDirect {
				if i, err := m.GetImage(direct); i != nil && err == nil {
					node := image(i)
					tree = append(tree, node)
				}
			}
			for _, indirect := range users.ImagesIndirect {
				if i, err := m.GetImage(indirect); i != nil && err == nil {
					node := image(i)
					tree = append(tree, node)
				}
			}
			for _, direct := range users.ContainersDirect {
				if c, err := m.GetContainer(direct); c != nil && err == nil {
					node := container(c)
					tree = append(tree, node)
				}
			}
			for _, indirect := range users.ContainersIndirect {
				if c, err := m.GetContainer(indirect); c != nil && err == nil {
					node := container(c)
					tree = append(tree, node)
				}
			}
			printTree(tree)
		} else {
			fmt.Printf("ID: %s\n", users.ID)
			fmt.Printf("Layer ID: %s\n", users.LayerID)
			for _, direct := range users.LayersDirect {
				fmt.Printf("Layer (Direct): %s\n", direct)
			}
			for _, indirect := range users.LayersIndirect {
				fmt.Printf("Layer (Indirect): %s\n", indirect)
			}
			for _, direct := range users.ImagesDirect {
				fmt.Printf("Image (Direct): %s\n", direct)
			}
			for _, indirect := range users.ImagesIndirect {
				fmt.Printf("Image (Indirect): %s\n", indirect)
			}
			for _, direct := range users.ContainersDirect {
				fmt.Printf("Container (Direct): %s\n", direct)
			}
			for _, indirect := range users.ContainersIndirect {
				fmt.Printf("Container (Indirect): %s\n", indirect)
			}
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"list-deps"},
		optionsHelp: "[LayerOrImageOrContainerNameOrID [...]]",
		usage:       "List referrers to a layer, image, or container",
		minArgs:     1,
		action:      listDeps,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&listLayersTree, []string{"-tree", "t"}, listLayersTree, "Use a tree")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
