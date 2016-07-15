package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/containers/storage/storage"
	"github.com/containers/storage/pkg/mflag"
)

var (
	MountLabel   = ""
	Name         = ""
	ID           = ""
	Image        = ""
	Layer        = ""
	Metadata     = ""
	MetadataFile = ""
	CreateRO     = false
)

func createLayer(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	parent := ""
	if len(args) > 0 {
		parent = args[0]
	}
	layer, err := m.CreateLayer(ID, parent, Name, MountLabel, !CreateRO)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if layer.Name != "" {
		fmt.Printf("%s\t%s\n", layer.ID, layer.Name)
	} else {
		fmt.Printf("%s\n", layer.ID)
	}
	return 0
}

func createImage(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
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
	image, err := m.CreateImage(ID, Name, args[0], Metadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if image.Name != "" {
		fmt.Printf("%s\t%s\n", image.ID, image.Name)
	} else {
		fmt.Printf("%s\n", image.ID)
	}
	return 0
}

func createContainer(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
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
	container, err := m.CreateContainer(ID, Name, args[0], Layer, Metadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if container.Name != "" {
		fmt.Printf("%s\t%s\n", container.ID, container.Name)
	} else {
		fmt.Printf("%s\n", container.ID)
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"create-layer"},
		optionsHelp: "[options [...]] [parentLayerNameOrID]",
		usage:       "Create a new layer",
		maxArgs:     1,
		action:      createLayer,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&MountLabel, []string{"-label", "l"}, "", "Mount Label")
			flags.StringVar(&Name, []string{"-name", "n"}, "", "Layer name")
			flags.StringVar(&ID, []string{"-id", "i"}, "", "Layer ID")
			flags.BoolVar(&CreateRO, []string{"-readonly", "r"}, false, "Mark as read-only")
		},
	})
	commands = append(commands, command{
		names:       []string{"create-image"},
		optionsHelp: "[options [...]] topLayerNameOrID",
		usage:       "Create a new image using layers",
		minArgs:     1,
		maxArgs:     1,
		action:      createImage,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&Name, []string{"-name", "n"}, "", "Image name")
			flags.StringVar(&ID, []string{"-id", "i"}, "", "Image ID")
			flags.StringVar(&Metadata, []string{"-metadata", "m"}, "", "Metadata")
			flags.StringVar(&MetadataFile, []string{"-metadata-file", "f"}, "", "Metadata File")
		},
	})
	commands = append(commands, command{
		names:       []string{"create-container"},
		optionsHelp: "[options [...]] parentImageNameOrID",
		usage:       "Create a new container from an image",
		minArgs:     1,
		maxArgs:     1,
		action:      createContainer,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&Name, []string{"-name", "n"}, "", "Container name")
			flags.StringVar(&ID, []string{"-id", "i"}, "", "Container ID")
			flags.StringVar(&Metadata, []string{"-metadata", "m"}, "", "Metadata")
			flags.StringVar(&MetadataFile, []string{"-metadata-file", "f"}, "", "Metadata File")
		},
	})
}
