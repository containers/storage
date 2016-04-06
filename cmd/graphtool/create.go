package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/distribution/digest"
	"github.com/docker/docker/image"
	"github.com/docker/docker/layer"
	"github.com/docker/docker/pkg/mflag"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/reference"
)

func create(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	var name layer.ChainID
	var rwID string
	var options []string

	lstore, err := m.GetLayerStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
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
	if len(args) > 0 {
		imageid, ref, err := reference.ParseIDOrReference(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing ID or reference %q: %v\n", args[0], err)
			roid, err := digest.ParseDigest(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing %q as digest: %v\n", args[0], err)
				return 1
			}
			name = layer.ChainID(roid)
			logrus.Debugf("Resolved %q to ID %q.", args[0], name)
		} else {
			var img *image.Image
			roid, err := digest.ParseDigest(imageid.String())
			if err != nil {
				associations := rstore.ReferencesByName(ref)
				if len(associations) == 0 {
					fmt.Fprintf(os.Stderr, "No image with name %s\n", ref.String())
				} else {
					logrus.Debugf("Resolved %q to name %q.", args[0], ref)
					for _, association := range associations {
						logrus.Debugf("Attempting to use ID %q.", association.ImageID)
						img, err = istore.Get(association.ImageID)
						if err == nil {
							break
						}
						fmt.Fprintf(os.Stderr, "No image with ID %s\n", association.ImageID)
					}
				}
			} else {
				logrus.Debugf("Resolved %q to ID %q.", args[0], roid)
				img, err = istore.Get(image.ID(roid))
			}
			if err != nil || img == nil {
				return 1
			}
			rootfs := img.RootFS
			if rootfs.Type != "layers" {
				fmt.Fprintf(os.Stderr, "Don't know how to deal with rootfs type %q, only layers.\n", rootfs.Type)
				return 1
			}
			if len(rootfs.DiffIDs) == 0 {
				fmt.Fprintf(os.Stderr, "No layers in this image.\n")
				return 1
			}
			name = rootfs.ChainID()
		}
		if len(args) > 1 {
			rwID = args[1]
		}
	}
	if rwID == "" {
		rwID = stringid.GenerateRandomID()
	}
	mountLabel := ""

	optsMap := make(map[string]string)
	for _, opt := range options {
		o := strings.SplitN(opt, "=", 2)
		if len(o) == 2 && o[0] != "" && o[1] != "" {
			optsMap[o[0]] = o[1]
		}
	}
	layer, err := lstore.CreateRWLayer(rwID, name, mountLabel, nil, optsMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating new layer from %q: %v\n", name, err)
		return 1
	}
	fmt.Printf("%s\n", layer.Name())
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"create"},
		optionsHelp: "imageID [name]",
		usage:       "Create a new read-write layer from an image",
		minArgs:     1,
		action:      create,
	})
}
