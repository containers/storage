package main

import (
	"fmt"
	"io"
	"os"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

var (
	paramLayerDataFile = ""
)

func listLayerBigData(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	layer, err := m.Layer(args[0])
	if err != nil {
		return 1, err
	}
	d, err := m.ListLayerBigData(layer.ID)
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		return outputJSON(d)
	}
	for _, name := range d {
		fmt.Printf("%s\n", name)
	}
	return 0, nil
}

func getLayerBigData(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	layer, err := m.Layer(args[0])
	if err != nil {
		return 1, err
	}
	output := os.Stdout
	if paramLayerDataFile != "" {
		f, err := os.Create(paramLayerDataFile)
		if err != nil {
			return 1, err
		}
		output = f
	}
	b, err := m.LayerBigData(layer.ID, args[1])
	if err != nil {
		return 1, err
	}
	if _, err := io.Copy(output, b); err != nil {
		return 1, err
	}
	output.Close()
	return 0, nil
}

func setLayerBigData(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	layer, err := m.Layer(args[0])
	if err != nil {
		return 1, err
	}
	input := os.Stdin
	if paramLayerDataFile != "" {
		f, err := os.Open(paramLayerDataFile)
		if err != nil {
			return 1, err
		}
		input = f
	}
	err = m.SetLayerBigData(layer.ID, args[1], input)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func layer(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	matched := []*storage.Layer{}
	for _, arg := range args {
		if layer, err := m.Layer(arg); err == nil {
			matched = append(matched, layer)
		}
	}
	if jsonOutput {
		if _, err := outputJSON(matched); err != nil {
			return 1, err
		}
	} else {
		for _, layer := range matched {
			fmt.Printf("ID: %s\n", layer.ID)
			for _, u := range layer.UIDMap {
				fmt.Printf("UID mapping: (container=%d, host=%d, size=%d)\n", u.ContainerID, u.HostID, u.Size)
			}
			for _, g := range layer.GIDMap {
				fmt.Printf("GID mapping: (container=%d, host=%d, size=%d)\n", g.ContainerID, g.HostID, g.Size)
			}
			if layer.Parent != "" {
				fmt.Printf("Parent: %s\n", layer.Parent)
			}
			for _, name := range layer.Names {
				fmt.Printf("Name: %s\n", name)
			}
			if layer.ReadOnly {
				fmt.Printf("Read Only: true\n")
			}
		}
	}
	if len(matched) != len(args) {
		return 1, nil
	}
	return 0, nil
}

func layerParentOwners(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	matched := []*storage.Layer{}
	for _, arg := range args {
		if layer, err := m.Layer(arg); err == nil {
			matched = append(matched, layer)
		}
	}
	for _, layer := range matched {
		uids, gids, err := m.LayerParentOwners(layer.ID)
		if err != nil {
			return 1, fmt.Errorf("LayerParentOwner: %+v", err)
		}
		if jsonOutput {
			mappings := struct {
				ID   string
				UIDs []int
				GIDs []int
			}{
				ID:   layer.ID,
				UIDs: uids,
				GIDs: gids,
			}
			if _, err := outputJSON(mappings); err != nil {
				return 1, err
			}
		} else {
			fmt.Printf("ID: %s\n", layer.ID)
			if len(uids) > 0 {
				fmt.Printf("UIDs: %v\n", uids)
			}
			if len(gids) > 0 {
				fmt.Printf("GIDs: %v\n", gids)
			}
		}
	}
	if len(matched) != len(args) {
		return 1, nil
	}
	return 0, nil
}

func init() {
	commands = append(commands,
		command{
			names:       []string{"layer"},
			optionsHelp: "[options [...]] layerNameOrID [...]",
			usage:       "Examine a layer",
			action:      layer,
			minArgs:     1,
			maxArgs:     -1,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			},
		},
		command{
			names:       []string{"layer-parent-owners"},
			optionsHelp: "[options [...]] layerNameOrID [...]",
			usage:       "Compute the set of unmapped parent UIDs and GIDs of the layer",
			action:      layerParentOwners,
			minArgs:     1,
			maxArgs:     -1,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			},
		},
		command{
			names:       []string{"list-layer-data", "listlayerdata"},
			optionsHelp: "[options [...]] layerNameOrID",
			usage:       "List data items that are attached to a layer",
			action:      listLayerBigData,
			minArgs:     1,
			maxArgs:     1,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			},
		},
		command{
			names:       []string{"get-layer-data", "getlayerdata"},
			optionsHelp: "[options [...]] layerID dataName",
			usage:       "Get data that is attached to a layer",
			action:      getLayerBigData,
			minArgs:     2,
			maxArgs:     2,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.StringVar(&paramLayerDataFile, []string{"-file", "f"}, paramLayerDataFile, "Write data to file")
			},
		},
		command{
			names:       []string{"set-layer-data", "setlayerdata"},
			optionsHelp: "[options [...]] layerID dataName",
			usage:       "Set data that is attached to a layer",
			action:      setLayerBigData,
			minArgs:     2,
			maxArgs:     2,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.StringVar(&paramLayerDataFile, []string{"-file", "f"}, paramLayerDataFile, "Read data from file")
			},
		},
	)
}
