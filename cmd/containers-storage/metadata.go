package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

var metadataQuiet = false

func metadata(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
	}
	metadataDict := make(map[string]string)
	missingAny := false
	for _, what := range args {
		if metadata, err := m.Metadata(what); err == nil {
			metadataDict[what] = strings.TrimSuffix(metadata, "\n")
		} else {
			missingAny = true
		}
	}
	if jsonOutput {
		if _, err := outputJSON(metadataDict); err != nil {
			return 1, err
		}
	} else {
		for _, what := range args {
			if metadataQuiet {
				fmt.Printf("%s\n", metadataDict[what])
			} else {
				fmt.Printf("%s: %s\n", what, metadataDict[what])
			}
		}
	}
	if missingAny {
		return 1, nil
	}
	return 0, nil
}

func setMetadata(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
	}
	if paramMetadataFile == "" && paramMetadata == "" {
		return 1, errors.New("no new metadata provided")
	}
	if paramMetadataFile != "" {
		f, err := os.Open(paramMetadataFile)
		if err != nil {
			return 1, err
		}
		b, err := io.ReadAll(f)
		if err != nil {
			return 1, err
		}
		paramMetadata = string(b)
	}
	if err := m.SetMetadata(args[0], paramMetadata); err != nil {
		return 1, err
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:       []string{"metadata"},
		optionsHelp: "[LayerOrImageOrContainerNameOrID [...]]",
		usage:       "Retrieve layer, image, or container metadata",
		minArgs:     1,
		maxArgs:     -1,
		action:      metadata,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&metadataQuiet, []string{"-quiet", "q"}, metadataQuiet, "Omit names and IDs")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"set-metadata", "setmetadata"},
		optionsHelp: "[options [...]] layerOrImageOrContainerNameOrID",
		usage:       "Set layer, image, or container metadata",
		minArgs:     1,
		maxArgs:     1,
		action:      setMetadata,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&paramMetadata, []string{"-metadata", "m"}, "", "Metadata")
			flags.StringVar(&paramMetadataFile, []string{"-metadata-file", "f"}, "", "Metadata File")
		},
	})
}
