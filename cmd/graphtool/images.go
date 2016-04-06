package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/mflag"
	"github.com/docker/docker/reference"
)

func imageMatch(i *image.Image, match string, names []reference.Named) bool {
	if match == "" || match == i.ImageID() || len(match) >= 12 && strings.HasPrefix(i.ImageID(), match) {
		return true
	}
	if match != "" {
		for _, name := range names {
			if strings.HasPrefix(name.FullName(), match) {
				return true
			}
			if strings.HasPrefix(name.Name(), match) {
				return true
			}
		}
	}
	return false
}

var listAllImages = false

func images(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	istore, err := m.GetImageStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	rstore, err := m.GetReferenceStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	images := istore.Map()
	for id, image := range images {
		references := rstore.References(id)
		if listAllImages || imageMatch(image, name, references) {
			fmt.Printf("%s\n", image.ImageID())
			for _, ref := range references {
				fmt.Printf("\t%s\n", ref.String())
			}
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"images"},
		optionsHelp: "[options [...]]",
		usage:       "List images",
		maxArgs:     1,
		action:      images,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&listAllImages, []string{"-all", "a"}, listAllImages, "List all images")
		},
	})
}
