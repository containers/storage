package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/chrootarchive"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/mflag"
)

var chownOptions = ""

func copyContent(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	var untarIDMappings *idtools.IDMappings
	var chownOpts *idtools.IDPair
	if len(args) < 1 {
		return 1, nil
	}
	if len(chownOptions) > 0 {
		uidStr, gidStr, ok := strings.Cut(chownOptions, ":")
		if !ok {
			gidStr = uidStr
		}
		uid, err := strconv.ParseUint(uidStr, 10, 32)
		if err != nil {
			return 1, fmt.Errorf("bad UID: %w", err)
		}
		gid, err := strconv.ParseUint(gidStr, 10, 32)
		if err != nil {
			return 1, fmt.Errorf("bad GID: %w", err)
		}
		chownOpts = &idtools.IDPair{UID: int(uid), GID: int(gid)}
	}
	target := args[len(args)-1]
	if strings.Contains(target, ":") {
		layer, targetRel, ok := strings.Cut(target, ":")
		if !ok {
			return 1, fmt.Errorf("error parsing target location %q: only one part", target)
		}
		targetLayer, err := m.Layer(layer)
		if err != nil {
			return 1, fmt.Errorf("error finding layer %q: %+v", layer, err)
		}
		untarIDMappings = idtools.NewIDMappingsFromMaps(targetLayer.UIDMap, targetLayer.GIDMap)
		targetMount, err := m.Mount(targetLayer.ID, targetLayer.MountLabel)
		if err != nil {
			return 1, fmt.Errorf("error mounting layer %q: %+v", targetLayer.ID, err)
		}
		target = filepath.Join(targetMount, targetRel)
		defer func() {
			if _, err := m.Unmount(targetLayer.ID, false); err != nil {
				fmt.Fprintf(os.Stderr, "error unmounting layer %q: %+v\n", targetLayer.ID, err)
				// Does not change the exit code
			}
		}()
	}
	args = args[:len(args)-1]
	for _, srcSpec := range args {
		var tarIDMappings *idtools.IDMappings
		source := srcSpec
		if layer, sourceRel, ok := strings.Cut(source, ":"); ok {
			sourceLayer, err := m.Layer(layer)
			if err != nil {
				return 1, fmt.Errorf("error finding layer %q: %+v", layer, err)
			}
			tarIDMappings = idtools.NewIDMappingsFromMaps(sourceLayer.UIDMap, sourceLayer.GIDMap)
			sourceMount, err := m.Mount(sourceLayer.ID, sourceLayer.MountLabel)
			if err != nil {
				return 1, fmt.Errorf("error mounting layer %q: %+v", sourceLayer.ID, err)
			}
			source = filepath.Join(sourceMount, sourceRel)
			defer func() {
				if _, err := m.Unmount(sourceLayer.ID, false); err != nil {
					fmt.Fprintf(os.Stderr, "error unmounting layer %q: %+v\n", sourceLayer.ID, err)
					// Does not change the exit code
				}
			}()
		}
		archiver := chrootarchive.NewArchiverWithChown(tarIDMappings, chownOpts, untarIDMappings)
		if err := archiver.CopyWithTar(source, target); err != nil {
			return 1, fmt.Errorf("error copying %q to %q: %+v", source, target, err)
		}
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:       []string{"copy"},
		usage:       "Copy files or directories into a layer, possibly from another layer",
		optionsHelp: "[options [...]] [sourceLayerNameOrID:]/path [...] targetLayerNameOrID:/path",
		minArgs:     2,
		maxArgs:     -1,
		action:      copyContent,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&chownOptions, []string{"-chown", "o"}, chownOptions, "Set owner on new copies")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
