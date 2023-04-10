package main

import (
	"os"
	"os/exec"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
	"github.com/containers/storage/pkg/unshare"
)

func unshareFn(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	unshare.MaybeReexecUsingUserNamespace(true)
	if len(args) == 0 {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		args = []string{shell}
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return 1, err
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:       []string{"unshare"},
		usage:       "Run a command in a user namespace",
		optionsHelp: "[command [...]]",
		minArgs:     0,
		maxArgs:     -1,
		action:      unshareFn,
	})
}
