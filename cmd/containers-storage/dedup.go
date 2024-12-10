package main

import (
	"fmt"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

var hashMethodArg = "crc"

func dedupCmd(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	var hashMethod storage.DedupHashMethod
	switch hashMethodArg {
	case "crc":
		hashMethod = storage.DedupHashCRC
	case "size":
		hashMethod = storage.DedupHashFileSize
	case "sha256":
		hashMethod = storage.DedupHashSHA256
	default:
		return 1, fmt.Errorf("invalid hash method: %s", hashMethodArg)
	}
	res, err := m.Dedup(storage.DedupArgs{
		Options: storage.DedupOptions{
			HashMethod: hashMethod,
		},
	})
	if err != nil {
		if jsonOutput {
			_, err2 := outputJSON(err)
			return 1, err2 // Note that err2 is usually nil
		}
		return 1, fmt.Errorf("%s: %+v", action, err)
	}
	if jsonOutput {
		return outputJSON(res)
	} else {
		fmt.Printf("Deduplicated %v bytes\n", res.Deduped)
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:   []string{"dedup"},
		usage:   "Dedup all images",
		minArgs: 0,
		maxArgs: 0,
		action:  dedupCmd,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			flags.StringVar(&hashMethodArg, []string{"-hash-method"}, hashMethodArg, "Specify the hash function to use to detect identical files")
		},
	})
}
