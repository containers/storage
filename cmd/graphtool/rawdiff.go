package main

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/mflag"
)

func rawChanges(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	to := args[0]
	from := ""
	if len(args) >= 2 {
		from = args[1]
	}
	changes, err := m.RawChanges(to, from)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
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

func rawDiff(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) < 2 {
		return 1
	}
	to := args[0]
	from := ""
	if len(args) >= 2 {
		from = args[1]
	}
	reader, err := m.RawDiff(to, from)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func rawApplyDiff(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	_, err := m.RawApplyDiff(args[0], os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	return 0
}

func rawDiffSize(flags *mflag.FlagSet, action string, m Mall, args []string) int {
	if len(args) < 1 {
		return 1
	}
	to := args[0]
	from := ""
	if len(args) >= 2 {
		from = args[1]
	}
	n, err := m.RawDiffSize(to, from)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	fmt.Printf("%d\n", n)
	return 0
}

func init() {
	commands = append(commands, command{
		names:       []string{"rawchanges"},
		usage:       "Compare two raw layers",
		optionsHelp: "[options [...]] layerNameOrID [referenceRawLayerNameOrID]",
		minArgs:     1,
		maxArgs:     2,
		action:      rawChanges,
	})
	commands = append(commands, command{
		names:       []string{"rawdiffsize"},
		usage:       "Compare two raw layers",
		optionsHelp: "[options [...]] layerNameOrID [referenceRawLayerNameOrID]",
		minArgs:     1,
		maxArgs:     2,
		action:      rawDiffSize,
	})
	commands = append(commands, command{
		names:       []string{"rawdiff"},
		usage:       "Compare two raw layers",
		optionsHelp: "[options [...]] layerNameOrID [referenceRawLayerNameOrID]",
		minArgs:     1,
		maxArgs:     2,
		action:      rawDiff,
	})
	commands = append(commands, command{
		names:       []string{"rawapplydiff"},
		optionsHelp: "[options [...]] layerNameOrID [referenceRawLayerNameOrID]",
		usage:       "Apply a diff to a raw layer",
		minArgs:     1,
		maxArgs:     1,
		action:      rawApplyDiff,
	})
}
