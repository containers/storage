package main

import (
	"fmt"
	"io"
	"os"

	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/storage"
)

var (
	applyDiffFile = ""
	diffFile      = ""
	diffGzip      = false
	diffBzip2     = false
	diffXz        = false
)

func changes(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	if len(args) < 1 {
		return 1
	}
	to := args[0]
	from := ""
	if len(args) >= 2 {
		from = args[1]
	}
	changes, err := m.Changes(from, to)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if jsonOutput {
		return jsonEncodeToStdout(changes)
	}

	for _, change := range changes {
		what := "?"
		switch change.Kind {
		case archive.ChangeAdd:
			what = "Add"
		case archive.ChangeModify:
			what = "Modify"
		case archive.ChangeDelete:
			what = "Delete"
		}
		fmt.Printf("%s %q\n", what, change.Path)
	}
	return 0
}

func diff(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	if len(args) < 1 {
		return 1
	}
	to := args[0]
	from := ""
	if len(args) >= 2 {
		from = args[1]
	}
	diffStream := io.Writer(os.Stdout)
	if diffFile != "" {
		f, err := os.Create(diffFile)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}

		diffStream = f
		defer f.Close()
	}
	reader, err := m.Diff(from, to)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	if diffGzip || diffBzip2 || diffXz {
		compression := archive.Uncompressed
		if diffGzip {
			compression = archive.Gzip
		} else if diffBzip2 {
			compression = archive.Bzip2
		} else if diffXz {
			compression = archive.Xz
		}
		compressor, compressErr := archive.CompressStream(diffStream, compression)
		if compressErr != nil {
			fmt.Fprintf(os.Stderr, "%v\n", compressErr)
			return 1
		}
		diffStream = compressor
		defer compressor.Close()
	}
	_, err = io.Copy(diffStream, reader)
	reader.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func applyDiff(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	if len(args) < 1 {
		return 1
	}
	diffStream := io.Reader(os.Stdin)
	if applyDiffFile != "" {
		f, err := os.Open(applyDiffFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		diffStream = f
		defer f.Close()
	}
	_, err := m.ApplyDiff(args[0], diffStream)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func diffSize(flags *mflag.FlagSet, action string, m storage.Store, args []string) int {
	if len(args) < 1 {
		return 1
	}
	to := args[0]
	from := ""
	if len(args) >= 2 {
		from = args[1]
	}
	n, err := m.DiffSize(from, to)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	fmt.Printf("%d\n", n)
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"changes"},
		usage:       "Compare two layers",
		optionsHelp: "[options [...]] layerNameOrID [referenceLayerNameOrID]",
		minArgs:     1,
		maxArgs:     2,
		action:      changes,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
		},
	})
	commands = append(commands, command{
		names:       []string{"diffsize", "diff-size"},
		usage:       "Compare two layers",
		optionsHelp: "[options [...]] layerNameOrID [referenceLayerNameOrID]",
		minArgs:     1,
		maxArgs:     2,
		action:      diffSize,
	})
	commands = append(commands, command{
		names:       []string{"diff"},
		usage:       "Compare two layers",
		optionsHelp: "[options [...]] layerNameOrID [referenceLayerNameOrID]",
		minArgs:     1,
		maxArgs:     2,
		action:      diff,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&diffFile, []string{"-file", "f"}, "", "Write to file instead of stdout")
			flags.BoolVar(&diffGzip, []string{"-gzip", "c"}, diffGzip, "Compress using gzip")
			flags.BoolVar(&diffBzip2, []string{"-bzip2", "-bz2", "b"}, diffBzip2, "Compress using bzip2")
			flags.BoolVar(&diffXz, []string{"-xz", "x"}, diffXz, "Compress using xz")
		},
	})
	commands = append(commands, command{
		names:       []string{"applydiff", "apply-diff"},
		optionsHelp: "[options [...]] layerNameOrID [referenceLayerNameOrID]",
		usage:       "Apply a diff to a layer",
		minArgs:     1,
		maxArgs:     1,
		action:      applyDiff,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.StringVar(&applyDiffFile, []string{"-file", "f"}, "", "Read from file instead of stdin")
		},
	})
}
