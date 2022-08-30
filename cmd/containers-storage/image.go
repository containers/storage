package main

import (
	"fmt"
	"io"
	"os"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
	digest "github.com/opencontainers/go-digest"
)

var (
	paramImageDataFile = ""
)

func image(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	matched := []*storage.Image{}
	for _, arg := range args {
		if image, err := m.Image(arg); err == nil {
			matched = append(matched, image)
		}
	}
	if jsonOutput {
		if _, err := outputJSON(matched); err != nil {
			return 1, err
		}
	} else {
		for _, image := range matched {
			fmt.Printf("ID: %s\n", image.ID)
			for _, name := range image.Names {
				fmt.Printf("Name: %s\n", name)
			}
			fmt.Printf("Top Layer: %s\n", image.TopLayer)
			for _, layerID := range image.MappedTopLayers {
				fmt.Printf("Top Layer: %s\n", layerID)
			}
			for _, digest := range image.Digests {
				fmt.Printf("Digest: %s\n", digest.String())
			}
			for _, name := range image.BigDataNames {
				fmt.Printf("Data: %s\n", name)
			}
			size, err := m.ImageSize(image.ID)
			if err != nil {
				fmt.Printf("Size unknown: %+v\n", err)
			} else {
				fmt.Printf("Size: %d\n", size)
			}
			if image.ReadOnly {
				fmt.Printf("Read Only: true\n")
			}
		}
	}
	if len(matched) != len(args) {
		return 1, nil
	}
	return 0, nil
}

func listImageBigData(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	image, err := m.Image(args[0])
	if err != nil {
		return 1, err
	}
	d, err := m.ListImageBigData(image.ID)
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

func getImageBigData(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	image, err := m.Image(args[0])
	if err != nil {
		return 1, err
	}
	output := os.Stdout
	if paramImageDataFile != "" {
		f, err := os.Create(paramImageDataFile)
		if err != nil {
			return 1, err
		}
		output = f
	}
	b, err := m.ImageBigData(image.ID, args[1])
	if err != nil {
		return 1, err
	}
	if _, err := output.Write(b); err != nil {
		return 1, err
	}
	output.Close()
	return 0, nil
}

func wrongManifestDigest(b []byte) (digest.Digest, error) {
	return digest.Canonical.FromBytes(b), nil
}

func getImageBigDataSize(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	image, err := m.Image(args[0])
	if err != nil {
		return 1, err
	}
	size, err := m.ImageBigDataSize(image.ID, args[1])
	if err != nil {
		return 1, err
	}
	fmt.Fprintf(os.Stdout, "%d\n", size)
	return 0, nil
}

func getImageBigDataDigest(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	image, err := m.Image(args[0])
	if err != nil {
		return 1, err
	}
	d, err := m.ImageBigDataDigest(image.ID, args[1])
	if err != nil {
		return 1, err
	}
	if err := d.Validate(); err != nil {
		return 1, err
	}
	fmt.Fprintf(os.Stdout, "%s\n", d.String())
	return 0, nil
}

func setImageBigData(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	image, err := m.Image(args[0])
	if err != nil {
		return 1, err
	}
	input := os.Stdin
	if paramImageDataFile != "" {
		f, err := os.Open(paramImageDataFile)
		if err != nil {
			return 1, err
		}
		input = f
	}
	b, err := io.ReadAll(input)
	if err != nil {
		return 1, err
	}
	err = m.SetImageBigData(image.ID, args[1], b, wrongManifestDigest)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func init() {
	commands = append(commands,
		command{
			names:       []string{"image"},
			optionsHelp: "[options [...]] imageNameOrID [...]",
			usage:       "Examine an image",
			action:      image,
			minArgs:     1,
			maxArgs:     -1,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			},
		},
		command{
			names:       []string{"list-image-data", "listimagedata"},
			optionsHelp: "[options [...]] imageNameOrID",
			usage:       "List data items that are attached to an image",
			action:      listImageBigData,
			minArgs:     1,
			maxArgs:     1,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			},
		},
		command{
			names:       []string{"get-image-data", "getimagedata"},
			optionsHelp: "[options [...]] imageNameOrID dataName",
			usage:       "Get data that is attached to an image",
			action:      getImageBigData,
			minArgs:     2,
			maxArgs:     2,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.StringVar(&paramImageDataFile, []string{"-file", "f"}, paramImageDataFile, "Write data to file")
			},
		},
		command{
			names:       []string{"get-image-data-size", "getimagedatasize"},
			optionsHelp: "[options [...]] imageNameOrID dataName",
			usage:       "Get size of data that is attached to an image",
			action:      getImageBigDataSize,
			minArgs:     2,
			maxArgs:     2,
		},
		command{
			names:       []string{"get-image-data-digest", "getimagedatadigest"},
			optionsHelp: "[options [...]] imageNameOrID dataName",
			usage:       "Get digest of data that is attached to an image",
			action:      getImageBigDataDigest,
			minArgs:     2,
			maxArgs:     2,
		},
		command{
			names:       []string{"set-image-data", "setimagedata"},
			optionsHelp: "[options [...]] imageNameOrID dataName",
			usage:       "Set data that is attached to an image (with sometimes wrong digest)",
			action:      setImageBigData,
			minArgs:     2,
			maxArgs:     2,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.StringVar(&paramImageDataFile, []string{"-file", "f"}, paramImageDataFile, "Read data from file")
			},
		})
}
