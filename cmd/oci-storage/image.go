package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

var (
	paramImageDataFile = ""
)

func image(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	images, err := m.Images()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	matched := []storage.Image{}
	for _, image := range images {
	nextImage:
		for _, arg := range args {
			if image.ID == arg {
				matched = append(matched, image)
				break nextImage
			}
			for _, name := range image.Names {
				if name == arg {
					matched = append(matched, image)
					break nextImage
				}
			}
		}
	}
	if jsonOutput {
		if jsonEncodeToStdout(matched) != 0 {
			return 1
		}
	} else {
		for _, image := range matched {
			fmt.Printf("ID: %s\n", image.ID)
			for _, name := range image.Names {
				fmt.Printf("Name: %s\n", name)
			}
			fmt.Printf("Top Layer: %s\n", image.TopLayer)
			for _, name := range image.BigDataNames {
				fmt.Printf("Data: %s\n", name)
			}
		}
	}
	if len(matched) != len(args) {
		return 1
	}
	return 0
}

func listImageBigData(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	image, err := m.GetImage(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	d, err := m.ListImageBigData(image.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	if jsonOutput {
		return jsonEncodeToStdout(d)
	}

	for _, name := range d {
		fmt.Printf("%s\n", name)
	}
	return 0
}

func getImageBigData(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	image, err := m.GetImage(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	output := os.Stdout
	if paramImageDataFile != "" {
		f, createErr := os.Create(paramImageDataFile)
		if createErr != nil {
			fmt.Fprintf(os.Stderr, "%v\n", createErr)
			return 1
		}
		output = f
	}
	b, err := m.GetImageBigData(image.ID, args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if _, err := output.Write(b); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	output.Close()
	return 0
}

func setImageBigData(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	image, err := m.GetImage(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	input := os.Stdin
	if paramImageDataFile != "" {
		f, openErr := os.Open(paramImageDataFile)
		if openErr != nil {
			fmt.Fprintf(os.Stderr, "%v\n", openErr)
			return 1
		}
		input = f
	}
	b, err := ioutil.ReadAll(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	err = m.SetImageBigData(image.ID, args[1], b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func init() {
	commands = append(commands,
		command{
			names:       []string{"image"},
			optionsHelp: "[options [...]] imageNameOrID [...]",
			usage:       "Examine an image",
			action:      image,
			minArgs:     1,
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
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.StringVar(&paramImageDataFile, []string{"-file", "f"}, paramImageDataFile, "Write data to file")
			},
		},
		command{
			names:       []string{"set-image-data", "setimagedata"},
			optionsHelp: "[options [...]] imageNameOrID dataName",
			usage:       "Set data that is attached to an image",
			action:      setImageBigData,
			minArgs:     2,
			addFlags: func(flags *mflag.FlagSet, cmd *command) {
				flags.StringVar(&paramImageDataFile, []string{"-file", "f"}, paramImageDataFile, "Read data from file")
			},
		})
}
