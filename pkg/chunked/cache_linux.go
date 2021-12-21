package chunked

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	storage "github.com/containers/storage"
	"github.com/containers/storage/pkg/chunked/internal"
)

func prepareOtherLayersCache(layersMetadata map[string][]internal.FileMetadata) map[string]map[string][]*internal.FileMetadata {
	maps := make(map[string]map[string][]*internal.FileMetadata)

	for layerID, v := range layersMetadata {
		r := make(map[string][]*internal.FileMetadata)
		for i := range v {
			if v[i].Digest != "" {
				r[v[i].Digest] = append(r[v[i].Digest], &v[i])
			}

		}
		maps[layerID] = r
	}
	return maps
}

func getLayersCache(store storage.Store) (map[string][]internal.FileMetadata, map[string]string, error) {
	allLayers, err := store.Layers()
	if err != nil {
		return nil, nil, err
	}

	layersMetadata := make(map[string][]internal.FileMetadata)
	layersTarget := make(map[string]string)
	for _, r := range allLayers {
		manifestReader, err := store.LayerBigData(r.ID, bigDataKey)
		if err != nil {
			continue
		}
		defer manifestReader.Close()
		manifest, err := ioutil.ReadAll(manifestReader)
		if err != nil {
			return nil, nil, fmt.Errorf("open manifest file for layer %q: %w", r.ID, err)
		}
		var toc internal.TOC
		if err := json.Unmarshal(manifest, &toc); err != nil {
			continue
		}
		layersMetadata[r.ID] = toc.Entries
		target, err := store.DifferTarget(r.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("get checkout directory layer %q: %w", r.ID, err)
		}
		layersTarget[r.ID] = target
	}

	return layersMetadata, layersTarget, nil
}

// findFileInOtherLayers finds the specified file in other layers.
// file is the file to look for.
// dirfd is an open file descriptor to the checkout root directory.
// layersMetadata contains the metadata for each layer in the storage.
// layersTarget maps each layer to its checkout on disk.
// useHardLinks defines whether the deduplication can be performed using hard links.
func findFileInOtherLayers(file *internal.FileMetadata, dirfd int, layersMetadata map[string]map[string][]*internal.FileMetadata, layersTarget map[string]string, useHardLinks bool) (bool, *os.File, int64, error) {
	// this is ugly, needs to be indexed
	for layerID, checksums := range layersMetadata {
		source, ok := layersTarget[layerID]
		if !ok {
			continue
		}
		files, found := checksums[file.Digest]
		if !found {
			continue
		}
		for _, candidate := range files {
			if candidate.Type != internal.TypeReg {
				continue
			}
			// check if it is a valid candidate to dedup file
			if useHardLinks && !canDedupMetadataWithHardLink(file, candidate) {
				continue
			}

			found, dstFile, written, err := copyFileFromOtherLayer(file, source, candidate, dirfd, useHardLinks)
			if found && err == nil {
				return found, dstFile, written, err
			}
		}
	}
	// If hard links deduplication was used and it has failed, try again without hard links.
	if useHardLinks {
		return findFileInOtherLayers(file, dirfd, layersMetadata, layersTarget, false)
	}
	return false, nil, 0, nil
}
