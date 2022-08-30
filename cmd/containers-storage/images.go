package main

import (
	"fmt"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
	digest "github.com/opencontainers/go-digest"
)

var (
	imagesQuiet = false
)

func images(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	images, err := m.Images()
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		return outputJSON(images)
	}
	for _, image := range images {
		fmt.Printf("%s\n", image.ID)
		if imagesQuiet {
			continue
		}
		for _, name := range image.Names {
			fmt.Printf("\tname: %s\n", name)
		}
		for _, digest := range image.Digests {
			fmt.Printf("\tdigest: %s\n", digest.String())
		}
		for _, name := range image.BigDataNames {
			fmt.Printf("\tdata: %s\n", name)
		}
	}
	return 0, nil
}

func imagesByDigest(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	images := []*storage.Image{}
	for _, arg := range args {
		d := digest.Digest(arg)
		if err := d.Validate(); err != nil {
			return 1, err
		}
		matched, err := m.ImagesByDigest(d)
		if err != nil {
			return 1, err
		}
		images = append(images, matched...)
	}
	if jsonOutput {
		return outputJSON(images)
	}
	for _, image := range images {
		fmt.Printf("%s\n", image.ID)
		if imagesQuiet {
			continue
		}
		for _, name := range image.Names {
			fmt.Printf("\tname: %s\n", name)
		}
		for _, digest := range image.Digests {
			fmt.Printf("\tdigest: %s\n", digest.String())
		}
		for _, name := range image.BigDataNames {
			fmt.Printf("\tdata: %s\n", name)
		}
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:       []string{"images"},
		optionsHelp: "[options [...]]",
		usage:       "List images",
		action:      images,
		minArgs:     0,
		maxArgs:     0,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			flags.BoolVar(&imagesQuiet, []string{"-quiet", "q"}, imagesQuiet, "Only print IDs")
		},
	})
	commands = append(commands, command{
		names:       []string{"images-by-digest"},
		optionsHelp: "[options [...]] DIGEST",
		usage:       "List images by digest",
		action:      imagesByDigest,
		minArgs:     1,
		maxArgs:     1,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			flags.BoolVar(&imagesQuiet, []string{"-quiet", "q"}, imagesQuiet, "Only print IDs")
		},
	})
}
