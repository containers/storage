package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/containers/storage"
	graphdriver "github.com/containers/storage/drivers"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/chunked"
	"github.com/containers/storage/pkg/chunked/compressor"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/containers/storage/pkg/mflag"
	digest "github.com/opencontainers/go-digest"
)

var (
	applyDiffFile    = ""
	diffFile         = ""
	diffUncompressed = false
	diffGzip         = false
	diffBzip2        = false
	diffXz           = false
)

func changes(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
	}
	to := args[0]
	from := ""
	if len(args) >= 2 {
		from = args[1]
	}
	changes, err := m.Changes(from, to)
	if err != nil {
		return 1, err
	}
	if jsonOutput {
		return outputJSON(changes)
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
	return 0, nil
}

func diff(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
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
			return 1, err
		}
		diffStream = f
		defer f.Close()
	}

	options := storage.DiffOptions{}
	if diffUncompressed || diffGzip || diffBzip2 || diffXz {
		c := archive.Uncompressed
		if diffGzip {
			c = archive.Gzip
		}
		if diffBzip2 {
			c = archive.Bzip2
		}
		if diffXz {
			c = archive.Xz
		}
		options.Compression = &c
	}

	reader, err := m.Diff(from, to, &options)
	if err != nil {
		return 1, err
	}
	_, err = io.Copy(diffStream, reader)
	reader.Close()
	if err != nil {
		return 1, err
	}
	return 0, nil
}

type fileFetcher struct {
	file *os.File
}

func sendFileParts(f *fileFetcher, chunks []chunked.ImageSourceChunk, streams chan io.ReadCloser, errors chan error) {
	defer close(streams)
	defer close(errors)

	for _, chunk := range chunks {
		l := io.NewSectionReader(f.file, int64(chunk.Offset), int64(chunk.Length))
		streams <- ioutils.NewReadCloserWrapper(l, func() error {
			return nil
		})
	}
}

func (f fileFetcher) GetBlobAt(chunks []chunked.ImageSourceChunk) (chan io.ReadCloser, chan error, error) {
	streams := make(chan io.ReadCloser)
	errs := make(chan error)
	go sendFileParts(&f, chunks, streams, errs)
	return streams, errs, nil
}

func applyDiffUsingStagingDirectory(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 2 {
		return 2, nil
	}

	layer := args[0]
	sourceDirectory := args[1]

	tOptions := archive.TarOptions{}
	tr, err := archive.TarWithOptions(sourceDirectory, &tOptions)
	if err != nil {
		return 1, err
	}
	defer tr.Close()

	tar, err := os.CreateTemp("", "layer-diff-tar-")
	if err != nil {
		return 1, err
	}
	defer os.Remove(tar.Name())
	defer tar.Close()

	// we go through the zstd:chunked compressor first so that it generates the metadata required to mount
	// a composefs image.

	metadata := make(map[string]string)
	compressor, err := compressor.ZstdCompressor(tar, metadata, nil)
	if err != nil {
		return 1, err
	}

	digesterCompressed := digest.Canonical.Digester()
	r := io.TeeReader(tr, digesterCompressed.Hash())

	if _, err := io.Copy(compressor, r); err != nil {
		return 1, err
	}
	if err := compressor.Close(); err != nil {
		return 1, err
	}

	size, err := tar.Seek(0, io.SeekCurrent)
	if err != nil {
		return 1, err
	}

	fetcher := fileFetcher{
		file: tar,
	}

	differ, err := chunked.GetDiffer(context.Background(), m, digesterCompressed.Digest(), size, metadata, &fetcher)
	if err != nil {
		return 1, err
	}

	var options graphdriver.ApplyDiffWithDifferOpts
	out, err := m.ApplyDiffWithDiffer("", &options, differ)
	if err != nil {
		return 1, err
	}

	applyStagedLayerArgs := storage.ApplyStagedLayerOptions{
		ID:          layer,
		DiffOutput:  out,
		DiffOptions: &options,
	}
	if _, err := m.ApplyStagedLayer(applyStagedLayerArgs); err != nil {
		m.CleanupStagedLayer(out)
		return 1, err
	}
	return 0, nil
}

func applyDiff(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
	}
	diffStream := io.Reader(os.Stdin)
	if applyDiffFile != "" {
		f, err := os.Open(applyDiffFile)
		if err != nil {
			return 1, err
		}
		diffStream = f
		defer f.Close()
	}
	_, err := m.ApplyDiff(args[0], diffStream)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func diffSize(flags *mflag.FlagSet, action string, m storage.Store, args []string) (int, error) {
	if len(args) < 1 {
		return 1, nil
	}
	to := args[0]
	from := ""
	if len(args) >= 2 {
		from = args[1]
	}
	n, err := m.DiffSize(from, to)
	if err != nil {
		return 1, err
	}
	fmt.Printf("%d\n", n)
	return 0, nil
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
			flags.BoolVar(&diffUncompressed, []string{"-uncompressed", "u"}, diffUncompressed, "Use no compression")
			flags.BoolVar(&diffGzip, []string{"-gzip", "c"}, diffGzip, "Compress using gzip")
			flags.BoolVar(&diffBzip2, []string{"-bzip2", "-bz2", "b"}, diffBzip2, "Compress using bzip2 (not currently supported)")
			flags.BoolVar(&diffXz, []string{"-xz", "x"}, diffXz, "Compress using xz (not currently supported)")
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
	commands = append(commands, command{
		names:       []string{"applydiff-using-staging-dir"},
		optionsHelp: "layerNameOrID directory",
		usage:       "Apply a diff to a layer using a staging directory",
		minArgs:     2,
		maxArgs:     2,
		action:      applyDiffUsingStagingDirectory,
	})
}
