package chunked

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	storage "github.com/containers/storage"
	"github.com/containers/storage/pkg/chunked/internal"
)

type layer struct {
	metadata map[string][]*internal.FileMetadata
	target   string
}

type layersCache struct {
	layers []layer
}

type findFileVisitor interface {
	VisitFile(file *internal.FileMetadata, target string) (bool, error)
}

func getLayersCache(store storage.Store) (*layersCache, error) {
	allLayers, err := store.Layers()
	if err != nil {
		return nil, err
	}

	cache := layersCache{}

	for _, r := range allLayers {
		manifestReader, err := store.LayerBigData(r.ID, bigDataKey)
		if err != nil {
			continue
		}
		defer manifestReader.Close()
		manifest, err := ioutil.ReadAll(manifestReader)
		if err != nil {
			return nil, fmt.Errorf("open manifest file for layer %q: %w", r.ID, err)
		}
		var toc internal.TOC
		if err := json.Unmarshal(manifest, &toc); err != nil {
			continue
		}
		target, err := store.DifferTarget(r.ID)
		if err != nil {
			return nil, fmt.Errorf("get checkout directory layer %q: %w", r.ID, err)
		}

		cache.addLayer(toc.Entries, target)
	}
	return &cache, nil
}

func (c *layersCache) addLayer(entries []internal.FileMetadata, target string) {
	r := make(map[string][]*internal.FileMetadata)
	for i := range entries {
		if entries[i].Digest != "" {
			r[entries[i].Digest] = append(r[entries[i].Digest], &entries[i])
		}

		// chunks do not use hard link dedup so keeping just one candidate is enough
		if entries[i].ChunkDigest != "" && len(r[entries[i].ChunkDigest]) == 0 {
			r[entries[i].ChunkDigest] = append(r[entries[i].ChunkDigest], &entries[i])
		}
	}
	l := layer{
		metadata: r,
		target:   target,
	}
	c.layers = append(c.layers, l)
}

// findFileInOtherLayers finds the specified file in other layers.
// file is the file to look for.
// visitor is the findFileVisitor to notify for each candidate found.
func (c *layersCache) findFileInOtherLayers(file *internal.FileMetadata, visitor findFileVisitor) error {
	for _, layer := range c.layers {
		files, found := layer.metadata[file.Digest]
		if !found {
			continue
		}
		for _, candidate := range files {
			if candidate.Type == internal.TypeReg {
				keepGoing, err := visitor.VisitFile(candidate, layer.target)
				if err != nil {
					return err
				}
				if !keepGoing {
					return nil
				}
			}
		}
	}
	return nil
}

func (c *layersCache) findChunkInOtherLayers(chunk *internal.FileMetadata) (string, string, int64) {
	for _, layer := range c.layers {
		entries, found := layer.metadata[chunk.ChunkDigest]
		if !found {
			continue
		}
		for _, candidate := range entries {
			if candidate.Type == internal.TypeChunk {
				return layer.target, candidate.Name, candidate.ChunkOffset
			}
			if candidate.Type == internal.TypeReg {
				return layer.target, candidate.Name, 0
			}
		}
	}
	return "", "", -1
}
