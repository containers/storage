package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

var metadataQuiet = false

func metadata(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	metadataDict := make(map[string]string)
	missingAny := false
	for _, what := range args {
		if container, err := m.GetContainer(what); err == nil {
			metadataDict[what] = strings.TrimSuffix(container.Metadata, "\n")
		} else if image, err := m.GetImage(what); err == nil {
			metadataDict[what] = strings.TrimSuffix(image.Metadata, "\n")
		} else {
			missingAny = true
		}
	}
	if jsonOutput {
		json.NewEncoder(os.Stdout).Encode(metadataDict)
	} else {
		for id, metadata := range metadataDict {
			if metadataQuiet {
				fmt.Printf("%s\n", metadata)
			} else {
				fmt.Printf("%s: %s\n", id, metadata)
			}
		}
	}
	if missingAny {
		return 1
	}
	return 0
}

func setMetadata(flags *mflag.FlagSet, action string, m storage.Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	if paramMetadataFile == "" && paramMetadata == "" {
		fmt.Fprintf(os.Stderr, "no new metadata provided\n")
		return 1
	}
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
	if err := m.SetMetadata(args[0], paramMetadata); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"metadata"},
		optionsHelp: "[ImageOrContainerNameOrID [...]]",
		usage:       "Retrieve image or container metadata",
		minArgs:     1,
		action:      metadata,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&metadataQuiet, []string{"-quiet", "q"}, metadataQuiet, "Omit names and IDs")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"set-metadata"},
		optionsHelp: "[options [...]] imageOrContainerNameOrID",
		usage:       "Set image or container metadata",
		minArgs:     1,
		maxArgs:     1,
		action:      setMetadata,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&paramMetadata, []string{"-metadata", "m"}, "", "Metadata")
			flags.StringVar(&paramMetadataFile, []string{"-metadata-file", "f"}, "", "Metadata File")
		},
	})
}
