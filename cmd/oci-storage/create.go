package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

var (
	paramMountLabel   = ""
	paramName         = ""
	paramID           = ""
	paramLayer        = ""
	paramMetadata     = ""
	paramMetadataFile = ""
	paramCreateRO     = false
)

func createLayer(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	parent := ""
	if len(args) > 0 {
		parent = args[0]
	}
	layer, err := m.CreateLayer(paramID, parent, paramName, paramMountLabel, !paramCreateRO)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(layer)
	} else {
		if layer.Name != "" {
			fmt.Printf("%s\t%s\n", layer.ID, layer.Name)
		} else {
			fmt.Printf("%s\n", layer.ID)
		}
	}
	return 0
}

func createImage(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if paramMetadataFile != "" {
		f, err := os.Open(paramMetadataFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		b, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		paramMetadata = string(b)
	}
	image, err := m.CreateImage(paramID, paramName, args[0], paramMetadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(image)
	} else {
		if image.Name != "" {
			fmt.Printf("%s\t%s\n", image.ID, image.Name)
		} else {
			fmt.Printf("%s\n", image.ID)
		}
	}
	return 0
}

func createContainer(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if paramMetadataFile != "" {
		f, err := os.Open(paramMetadataFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		b, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		paramMetadata = string(b)
	}
	container, err := m.CreateContainer(paramID, paramName, args[0], paramLayer, paramMetadata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(container)
	} else {
		if container.Name != "" {
			fmt.Printf("%s\t%s\n", container.ID, container.Name)
		} else {
			fmt.Printf("%s\n", container.ID)
		}
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
			flags.StringVar(&paramMountLabel, []string{"-label", "l"}, "", "Mount Label")
			flags.StringVar(&paramName, []string{"-name", "n"}, "", "Layer name")
			flags.StringVar(&paramID, []string{"-id", "i"}, "", "Layer ID")
			flags.BoolVar(&paramCreateRO, []string{"-readonly", "r"}, false, "Mark as read-only")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "prefer JSON output")
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
			flags.StringVar(&paramName, []string{"-name", "n"}, "", "Image name")
			flags.StringVar(&paramID, []string{"-id", "i"}, "", "Image ID")
			flags.StringVar(&paramMetadata, []string{"-metadata", "m"}, "", "Metadata")
			flags.StringVar(&paramMetadataFile, []string{"-metadata-file", "f"}, "", "Metadata File")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "prefer JSON output")
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
			flags.StringVar(&paramName, []string{"-name", "n"}, "", "Container name")
			flags.StringVar(&paramID, []string{"-id", "i"}, "", "Container ID")
			flags.StringVar(&paramMetadata, []string{"-metadata", "m"}, "", "Metadata")
			flags.StringVar(&paramMetadataFile, []string{"-metadata-file", "f"}, "", "Metadata File")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "prefer JSON output")
		},
	})
}
