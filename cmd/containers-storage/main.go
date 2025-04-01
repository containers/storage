package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/storage"
	"github.com/containers/storage/internal/opts"
	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/unshare"
	"github.com/containers/storage/types"
	"github.com/sirupsen/logrus"
)

type command struct {
	names       []string
	optionsHelp string
	minArgs     int
	maxArgs     int
	usage       string
	addFlags    func(*mflag.FlagSet, *command)
	action      func(*mflag.FlagSet, string, storage.Store, []string) (int, error)
}

var (
	commands   = []command{}
	jsonOutput = false
	force      = false
)

func main() {
	if reexec.Init() {
		return
	}

	options := types.StoreOptions{}
	debug := false
	doUnshare := false

	makeFlags := func(command string, eh mflag.ErrorHandling) *mflag.FlagSet {
		flags := mflag.NewFlagSet(command, eh)
		flags.StringVar(&options.RunRoot, []string{"-run", "R"}, options.RunRoot, "Root of the runtime state tree")
		flags.StringVar(&options.GraphRoot, []string{"-graph", "g"}, options.GraphRoot, "Root of the storage tree")
		flags.StringVar(&options.ImageStore, []string{"-image-store"}, options.ImageStore, "Root of the separate image store")
		flags.BoolVar(&options.TransientStore, []string{"-transient-store"}, options.TransientStore, "Transient store")
		flags.StringVar(&options.GraphDriverName, []string{"-storage-driver", "s"}, options.GraphDriverName, "Storage driver to use ($STORAGE_DRIVER)")
		flags.Var(opts.NewListOptsRef(&options.GraphDriverOptions, nil), []string{"-storage-opt"}, "Set storage driver options ($STORAGE_OPTS)")
		flags.BoolVar(&debug, []string{"-debug", "D"}, debug, "Print debugging information")
		flags.BoolVar(&doUnshare, []string{"-unshare", "U"}, unshare.IsRootless(), fmt.Sprintf("Run in a user namespace (default %t)", unshare.IsRootless()))
		return flags
	}

	flags := makeFlags("containers-storage", mflag.ContinueOnError)
	flags.Usage = func() {
		fmt.Printf("Usage: containers-storage command [options [...]]\n\n")
		fmt.Printf("Commands:\n\n")
		for _, command := range commands {
			fmt.Printf("  %-30s%s\n", command.names[0], command.usage)
		}
		fmt.Printf("\nOptions:\n")
		flags.PrintDefaults()
	}

	if len(os.Args) < 2 {
		flags.Usage()
		os.Exit(1)
	}
	if err := flags.ParseFlags(os.Args[1:], true); err != nil {
		fmt.Printf("%v while parsing arguments (1)\n", err)
		flags.Usage()
		os.Exit(1)
	}
	if options.GraphRoot == "" && options.RunRoot == "" && options.GraphDriverName == "" && len(options.GraphDriverOptions) == 0 {
		options, _ = types.DefaultStoreOptions()
	}
	args := flags.Args()
	if len(args) < 1 {
		flags.Usage()
		os.Exit(1)
		return
	}
	cmd := args[0]

	for _, command := range commands {
		for _, name := range command.names {
			if cmd == name {
				flags := makeFlags(cmd, mflag.ExitOnError)
				if command.addFlags != nil {
					command.addFlags(flags, &command)
				}
				flags.Usage = func() {
					fmt.Printf("Usage: containers-storage %s %s\n\n", cmd, command.optionsHelp)
					fmt.Printf("%s\n", command.usage)
					fmt.Printf("\nOptions:\n")
					flags.PrintDefaults()
				}
				if err := flags.ParseFlags(args[1:], false); err != nil {
					fmt.Printf("%v while parsing arguments (3)", err)
					flags.Usage()
					os.Exit(1)
				}
				args = flags.Args()
				if command.minArgs != 0 && len(args) < command.minArgs {
					fmt.Printf("%s: more arguments required.\n", cmd)
					flags.Usage()
					os.Exit(1)
				}
				if command.maxArgs >= 0 && command.maxArgs < command.minArgs {
					panic(fmt.Sprintf("command %v requires more args (%d) than it allows (%d)", command.names, command.minArgs, command.maxArgs))
				}
				if command.maxArgs >= 0 && len(args) > command.maxArgs {
					fmt.Printf("%s: too many arguments (%s).\n", cmd, args)
					flags.Usage()
					os.Exit(1)
				}
				if doUnshare {
					unshare.MaybeReexecUsingUserNamespace(true)
				}
				if debug {
					logrus.SetLevel(logrus.DebugLevel)
					logrus.Debugf("Root: %s", options.GraphRoot)
					logrus.Debugf("Run Root: %s", options.RunRoot)
					logrus.Debugf("Driver Name: %s", options.GraphDriverName)
					logrus.Debugf("Driver Options: %s", options.GraphDriverOptions)
				} else {
					logrus.SetLevel(logrus.ErrorLevel)
				}
				store, err := storage.GetStore(options)
				if err != nil {
					fmt.Printf("error initializing: %+v\n", err)
					os.Exit(1)
				}
				store.Free()
				res, err := command.action(flags, cmd, store, args)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%+v\n", err)
				}
				os.Exit(res)
				break
			}
		}
	}
	fmt.Printf("%s: unrecognized command.\n", cmd)
	os.Exit(1)
}

// outputJSON formats its input as JSON to stdout, and returns values suitable
// for directly returning from command.action
func outputJSON(data any) (int, error) {
	if err := json.NewEncoder(os.Stdout).Encode(data); err != nil {
		return 1, err
	}
	return 0, nil
}
