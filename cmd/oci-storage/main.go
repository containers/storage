package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/containers/storage/opts"
	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/storage"
)

type command struct {
	names       []string
	optionsHelp string
	minArgs     int
	maxArgs     int
	usage       string
	addFlags    func(*mflag.FlagSet, *command)
	action      func(*mflag.FlagSet, string, storage.Store, []string) int
}

var (
	commands   = []command{}
	jsonOutput = false
)

func main() {
	if reexec.Init() {
		return
	}

	options := storage.DefaultStoreOptions
	g := options.GraphMap[0]
	debug := false
	ro_graph := opts.NewListOpts(nil)

	makeFlags := func(command string, eh mflag.ErrorHandling) *mflag.FlagSet {
		flags := mflag.NewFlagSet(command, eh)
		flags.StringVar(&options.RunRoot, []string{"-run", "R"}, options.RunRoot, "Root of the runtime state tree")
		flags.StringVar(&g.Root, []string{"-graph", "g"}, g.Root, "Root of the storage tree")
		flags.StringVar(&g.DriverName, []string{"-storage-driver", "s"}, g.DriverName, "Storage driver to use ($STORAGE_DRIVER)")
		flags.Var(opts.NewListOptsRef(&g.DriverOptions, nil), []string{"-storage-opt"}, "Set storage driver options ($STORAGE_OPTS)")
		flags.BoolVar(&debug, []string{"-debug", "D"}, debug, "Print debugging information")
		flags.Var(&ro_graph, []string{"-graph"}, "Additional Read/Only Root image tree")
		return flags
	}

	flags := makeFlags("oci-storage", mflag.ContinueOnError)
	flags.Usage = func() {
		fmt.Printf("Usage: oci-storage command [options [...]]\n\n")
		fmt.Printf("Commands:\n\n")
		for _, command := range commands {
			fmt.Printf("  %-22s%s\n", command.names[0], command.usage)
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

	args := flags.Args()
	if len(args) < 1 {
		flags.Usage()
		os.Exit(1)
		return
	}
	cmd := args[0]

	for _, root := range ro_graph.GetAll() {
		var g storage.GraphDriver
		g.DriverName = "overlay"
		g.Root = root
		g.DriverOptions = nil
		options.GraphMap = append(options.GraphMap, g)
	}

	for _, command := range commands {
		for _, name := range command.names {
			if cmd == name {
				flags := makeFlags(cmd, mflag.ExitOnError)
				if command.addFlags != nil {
					command.addFlags(flags, &command)
				}
				flags.Usage = func() {
					fmt.Printf("Usage: oci-storage %s %s\n\n", cmd, command.optionsHelp)
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
				if command.maxArgs != 0 && len(args) > command.maxArgs {
					fmt.Printf("%s: too many arguments (%s).\n", cmd, args)
					flags.Usage()
					os.Exit(1)
				}
				if debug {
					logrus.SetLevel(logrus.DebugLevel)
					logrus.Debugf("RunRoot: %s", options.RunRoot)
					logrus.Debugf("GraphRoot: %s", options.GraphMap[0].Root)
					logrus.Debugf("GraphDriverName: %s", options.GraphMap[0].DriverName)
					logrus.Debugf("GraphDriverOptions: %s", options.GraphMap[0].DriverOptions)
				} else {
					logrus.SetLevel(logrus.ErrorLevel)
				}
				store, err := storage.GetStore(options)
				if err != nil {
					fmt.Printf("error initializing: %v\n", err)
					os.Exit(1)
				}
				os.Exit(command.action(flags, cmd, store, args))
				break
			}
		}
	}
	fmt.Printf("%s: unrecognized command.\n", cmd)
	os.Exit(1)
}
