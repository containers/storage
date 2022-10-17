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

var (
	chownOptions = ""
)

func copyContent(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	var untarIDMappings *idtools.IDMappings
	var chownOpts *idtools.IDPair
	if len(args) < 1 {
		return 1, nil
	}
	if len(chownOptions) > 0 {
		chownParts := strings.SplitN(chownOptions, ":", 2)
		if len(chownParts) == 1 {
			chownParts = append(chownParts, chownParts[0])
		}
		uid, err := strconv.ParseUint(chownParts[0], 10, 32)
		if err != nil {
			return 1, fmt.Errorf("error %q as a numeric UID: %v", chownParts[0], err)
		}
		gid, err := strconv.ParseUint(chownParts[1], 10, 32)
		if err != nil {
			return 1, fmt.Errorf("error %q as a numeric GID: %v", chownParts[1], err)
		}
		chownOpts = &idtools.IDPair{UID: int(uid), GID: int(gid)}
	}
	target := args[len(args)-1]
	if strings.Contains(target, ":") {
		targetParts := strings.SplitN(target, ":", 2)
		if len(targetParts) != 2 {
			return 1, fmt.Errorf("error parsing target location %q: only one part", target)
		}
		targetLayer, err := m.Layer(targetParts[0])
		if err != nil {
			return 1, fmt.Errorf("error finding layer %q: %+v", targetParts[0], err)
		}
		untarIDMappings = idtools.NewIDMappingsFromMaps(targetLayer.UIDMap, targetLayer.GIDMap)
		targetMount, err := m.Mount(targetLayer.ID, targetLayer.MountLabel)
		if err != nil {
			return 1, fmt.Errorf("error mounting layer %q: %+v", targetLayer.ID, err)
		}
		target = filepath.Join(targetMount, targetParts[1])
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
		if strings.Contains(source, ":") {
			sourceParts := strings.SplitN(source, ":", 2)
			if len(sourceParts) != 2 {
				return 1, fmt.Errorf("error parsing source location %q: only one part", source)
			}
			sourceLayer, err := m.Layer(sourceParts[0])
			if err != nil {
				return 1, fmt.Errorf("error finding layer %q: %+v", sourceParts[0], err)
			}
			tarIDMappings = idtools.NewIDMappingsFromMaps(sourceLayer.UIDMap, sourceLayer.GIDMap)
			sourceMount, err := m.Mount(sourceLayer.ID, sourceLayer.MountLabel)
			if err != nil {
				return 1, fmt.Errorf("error mounting layer %q: %+v", sourceLayer.ID, err)
			}
			source = filepath.Join(sourceMount, sourceParts[1])
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
		action:      copyContent,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&chownOptions, []string{"-chown", ""}, chownOptions, "Set owner on new copies")
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
}
