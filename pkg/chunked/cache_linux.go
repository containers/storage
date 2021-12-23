package chunked

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	storage "github.com/containers/storage"
	"github.com/containers/storage/pkg/chunked/internal"
)

type layer struct {
	id       string
	metadata map[string][]*internal.FileMetadata
	target   string
}

type layersCache struct {
	layers  []layer
	refs    int
	store   storage.Store
	mutex   sync.Mutex
	created time.Time
}

type findFileVisitor interface {
	VisitFile(file *internal.FileMetadata, target string) (bool, error)
}

var cacheMutex sync.Mutex
var cache *layersCache

func (c *layersCache) release() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	c.refs--
	if c.refs == 0 {
		cache = nil
	}
}

func getLayersCacheRef(store storage.Store) *layersCache {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	if cache != nil && cache.store == store && time.Since(cache.created).Minutes() < 10 {
		cache.refs++
		return cache
	}
	cache := &layersCache{
		store:   store,
		refs:    1,
		created: time.Now(),
	}
	return cache
}

func getLayersCache(store storage.Store) (*layersCache, error) {
	c := getLayersCacheRef(store)

	if err := c.load(); err != nil {
		c.release()
		return nil, err
	}
	return c, nil
}

func (c *layersCache) load() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	allLayers, err := c.store.Layers()
	if err != nil {
		return err
	}
	existingLayers := make(map[string]string)
	for _, r := range c.layers {
		existingLayers[r.id] = r.target
	}

	currentLayers := make(map[string]string)
	for _, r := range allLayers {
		currentLayers[r.ID] = r.ID
		if _, found := existingLayers[r.ID]; found {
			continue
		}
		manifestReader, err := c.store.LayerBigData(r.ID, bigDataKey)
		if err != nil {
			continue
		}
		defer manifestReader.Close()
		manifest, err := ioutil.ReadAll(manifestReader)
		if err != nil {
			return fmt.Errorf("open manifest file for layer %q: %w", r.ID, err)
		}
		var toc internal.TOC
		if err := json.Unmarshal(manifest, &toc); err != nil {
			continue
		}
		target, err := c.store.DifferTarget(r.ID)
		if err != nil {
			return fmt.Errorf("get checkout directory layer %q: %w", r.ID, err)
		}

		c.addLayer(r.ID, toc.Entries, target)
	}

	var newLayers []layer
	for _, l := range c.layers {
		if _, found := currentLayers[l.id]; found {
			newLayers = append(newLayers, l)
		}
	}
	c.layers = newLayers

	return nil
}

func (c *layersCache) addLayer(id string, entries []internal.FileMetadata, target string) {
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
		id:       id,
		metadata: r,
		target:   target,
	}
	c.layers = append(c.layers, l)
}

// findFileInOtherLayers finds the specified file in other layers.
// file is the file to look for.
// visitor is the findFileVisitor to notify for each candidate found.
func (c *layersCache) findFileInOtherLayers(file *internal.FileMetadata, visitor findFileVisitor) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
