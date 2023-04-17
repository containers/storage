package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/mflag"
)

var (
	quickCheck, repair, forceRepair bool
	maximumUnreferencedLayerAge     string
)

func check(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if forceRepair {
		repair = true
	}
	defer func() {
		if _, err := m.Shutdown(true); err != nil {
			fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
		}
	}()
	checkOptions := storage.CheckEverything()
	if quickCheck {
		checkOptions = storage.CheckMost()
	}
	if maximumUnreferencedLayerAge != "" {
		age, err := time.ParseDuration(maximumUnreferencedLayerAge)
		if err != nil {
			return 1, err
		}
		checkOptions.LayerUnreferencedMaximumAge = &age
	}
	report, err := m.Check(checkOptions)
	if err != nil {
		return 1, err
	}
	outputNonJSON := func(report storage.CheckReport) {
		for id, errs := range report.Layers {
			if len(errs) > 0 {
				fmt.Fprintf(os.Stdout, "layer %s:\n", id)
			}
			for _, err := range errs {
				fmt.Fprintf(os.Stdout, " %v\n", err)
			}
		}
		for id, errs := range report.ROLayers {
			if len(errs) > 0 {
				fmt.Fprintf(os.Stdout, "read-only layer %s:\n", id)
			}
			for _, err := range errs {
				fmt.Fprintf(os.Stdout, " %v\n", err)
			}
		}
		for id, errs := range report.Images {
			if len(errs) > 0 {
				fmt.Fprintf(os.Stdout, "image %s:\n", id)
			}
			for _, err := range errs {
				fmt.Fprintf(os.Stdout, " %v\n", err)
			}
		}
		for id, errs := range report.ROImages {
			if len(errs) > 0 {
				fmt.Fprintf(os.Stdout, "read-only image %s:\n", id)
			}
			for _, err := range errs {
				fmt.Fprintf(os.Stdout, " %v\n", err)
			}
		}
		for id, errs := range report.Containers {
			if len(errs) > 0 {
				fmt.Fprintf(os.Stdout, "container %s:\n", id)
			}
			for _, err := range errs {
				fmt.Fprintf(os.Stdout, " %v\n", err)
			}
		}
	}

	if jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			return 1, err
		}
	} else {
		outputNonJSON(report)
	}

	if !repair {
		if len(report.Layers) > 0 || len(report.ROLayers) > 0 || len(report.Images) > 0 || len(report.ROImages) > 0 || len(report.Containers) > 0 {
			return 1, fmt.Errorf("%d layer errors, %d read-only layer errors, %d image errors, %d read-only image errors, %d container errors", len(report.Layers), len(report.ROLayers), len(report.Images), len(report.ROImages), len(report.Containers))
		}
	} else {
		options := storage.RepairOptions{
			RemoveContainers: forceRepair,
		}
		if errs := m.Repair(report, &options); len(errs) != 0 {
			if jsonOutput {
				if err := json.NewEncoder(os.Stdout).Encode(errs); err != nil {
					return 1, err
				}
			} else {
				for _, err := range errs {
					fmt.Fprintf(os.Stderr, "%v\n", err)
				}
			}
			return 1, errs[0]
		}
		if len(report.ROLayers) > 0 || len(report.ROImages) > 0 || (!options.RemoveContainers && len(report.Containers) > 0) {
			var err error
			if options.RemoveContainers {
				err = fmt.Errorf("%d read-only layer errors, %d read-only image errors", len(report.ROLayers), len(report.ROImages))
			} else {
				err = fmt.Errorf("%d read-only layer errors, %d read-only image errors, %d container errors", len(report.ROLayers), len(report.ROImages), len(report.Containers))
			}
			return 1, err
		}
	}
	return 0, nil
}

func init() {
	commands = append(commands, command{
		names:   []string{"check"},
		usage:   "Check storage consistency",
		minArgs: 0,
		maxArgs: 0,
		action:  check,
		addFlags: func(flags *mflag.FlagSet, cmd *command) {
			flags.BoolVar(&jsonOutput, []string{"-json", "j"}, jsonOutput, "Prefer JSON output")
			flags.StringVar(&maximumUnreferencedLayerAge, []string{"-max", "m"}, "24h", "Maximum allowed age for unreferenced layers")
			flags.BoolVar(&repair, []string{"-repair", "r"}, repair, "Remove damaged images and layers")
			flags.BoolVar(&forceRepair, []string{"-force", "f"}, forceRepair, "Remove damaged containers")
			flags.BoolVar(&quickCheck, []string{"-quick", "q"}, quickCheck, "Perform only quick checks")
		},
	})
}
