package main

import (
	"fmt"
	"os"

	"github.com/docker/docker/container"
	"github.com/docker/docker/layer"
	"github.com/docker/docker/pkg/mflag"
)

func getRWLayer(m Mall, name string) (layer.RWLayer, *container.Container, error) {
	pstore, err := m.GetPetStore()
	if err != nil {
		return nil, nil, err
	}
	cstore, err := m.GetContainerStore()
	if err != nil {
		return nil, nil, err
	}
	layerStore, err := m.GetLayerStore()
	if err != nil {
		return nil, nil, err
	}
	pets, err := pstore.List()
	if err != nil {
		return nil, nil, err
	}
	for _, pet := range pets {
		if petMatch(pet, name) {
			return pet.Layer(), nil, nil
		}
	}
	for _, container := range cstore.List() {
		if containerMatch(container, name) {
			layer, err := layerStore.GetRWLayer(container.ID)
			if err != nil {
				return nil, nil, err
			}
			return layer, container, nil
		}
	}
	layer, err := layerStore.GetRWLayer(name)
	if err != nil {
		return nil, nil, err
	}
	if layer == nil {
		return nil, nil, noMatchingContainerError
	}
	return layer, nil, nil
}

func mount(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	for _, arg := range args {
		layer, container, err := getRWLayer(m, arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s while mounting %s\n", err, arg)
			return 1
		}
		mountLabel := ""
		if container != nil {
			mountLabel = container.GetMountLabel()
		}
		result, err := layer.Mount(mountLabel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s while mounting %s\n", err, arg)
			return 1
		}
		fmt.Printf("%s\n", result)
	}
	return 0
}

func unmount(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	for _, arg := range args {
		layer, _, err := getRWLayer(m, arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s while unmounting %s\n", err, arg)
			return 1
		}
		err = layer.Unmount()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s while unmounting %s\n", err, arg)
			return 1
		}
	}
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"mount"},
		optionsHelp: "petNameOrContainerName",
		usage:       "Mount a read-write layer",
		minArgs:     1,
		action:      mount,
	})
	commands = append(commands, command{
		names:       []string{"unmount", "umount"},
		optionsHelp: "petNameOrContainerName",
		usage:       "Unmount a read-write layer",
		minArgs:     1,
		action:      unmount,
	})
}
