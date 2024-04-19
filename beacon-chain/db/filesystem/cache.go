package filesystem

import (
	"context"
	"sync"
	"time"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/spf13/afero"
)

// blobIndexMask is a bitmask representing the set of blob indices that are currently set.
type blobIndexMask [fieldparams.MaxBlobsPerBlock]bool

// BlobStorageSummary represents cached information about the BlobSidecars on disk for each root the cache knows about.
type BlobStorageSummary struct {
	epoch primitives.Epoch
	mask  blobIndexMask
}

// HasIndex returns true if the BlobSidecar at the given index is available in the filesystem.
func (s BlobStorageSummary) HasIndex(idx uint64) bool {
	// Protect from panic, but assume callers are sophisticated enough to not need an error telling them they have an invalid idx.
	if idx >= fieldparams.MaxBlobsPerBlock {
		return false
	}
	return s.mask[idx]
}

// AllAvailable returns true if we have all blobs for all indices from 0 to count-1.
func (s BlobStorageSummary) AllAvailable(count int) bool {
	if count > fieldparams.MaxBlobsPerBlock {
		return false
	}
	for i := 0; i < count; i++ {
		if !s.mask[i] {
			return false
		}
	}
	return true
}

// BlobStorageSummarizer can be used to receive a summary of metadata about blobs on disk for a given root.
// The BlobStorageSummary can be used to check which indices (if any) are available for a given block by root.
type BlobStorageSummarizer interface {
	Summary(root [32]byte) BlobStorageSummary
}

type blobStorageCache struct {
	mu       sync.RWMutex
	nBlobs   float64
	cache    map[[32]byte]BlobStorageSummary
	ready    chan struct{}
	warmDone bool
	warmer   *cacheWarmer
}

var _ BlobStorageSummarizer = &blobStorageCache{}

func newBlobStorageCache() *blobStorageCache {
	return &blobStorageCache{
		cache:  make(map[[32]byte]BlobStorageSummary),
		warmer: &cacheWarmer{ready: make(chan struct{})},
	}
}

// Summary returns the BlobStorageSummary for `root`. The BlobStorageSummary can be used to check for the presence of
// BlobSidecars based on Index.
func (s *blobStorageCache) Summary(root [32]byte) BlobStorageSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache[root]
}

func (s *blobStorageCache) ensure(key [32]byte, epoch primitives.Epoch, idx uint64) error {
	if idx >= fieldparams.MaxBlobsPerBlock {
		return errIndexOutOfBounds
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.cache[key]
	v.epoch = epoch
	if !v.mask[idx] {
		s.updateMetrics(1)
	}
	v.mask[idx] = true
	s.cache[key] = v
	return nil
}

func (s *blobStorageCache) epoch(key [32]byte) (primitives.Epoch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.cache[key]
	if !ok {
		return 0, false
	}
	return v.epoch, ok
}

func (s *blobStorageCache) evict(key [32]byte) int {
	deleted := 0
	s.mu.Lock()
	v, ok := s.cache[key]
	if ok {
		for i := range v.mask {
			if v.mask[i] {
				deleted += 1
			}
		}
	}
	delete(s.cache, key)
	s.mu.Unlock()
	if deleted > 0 {
		s.updateMetrics(-float64(deleted))
	}
	return deleted
}

func (s *blobStorageCache) updateMetrics(delta float64) {
	s.nBlobs += delta
	blobDiskCount.Set(s.nBlobs)
	blobDiskSize.Set(s.nBlobs * bytesPerSidecar)
}

func (p *blobStorageCache) waitForReady(ctx context.Context) error {
	select {
	case <-p.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *blobStorageCache) warm(fs afero.Fs) error {
	return c.warmer.warm(c, fs)
}

func (w *cacheWarmer) warm(cache *blobStorageCache, fs afero.Fs) error {
	w.mu.Lock()
	start := time.Now()
	defer func() {
		w.mu.Unlock()
		log.WithField("duration", time.Since(start).String()).Debug("Warmed up pruner cache")
	}()
	if w.warmed {
		return nil
	}

	layout, err := detectLayout(fs)
	if err != nil {
		return err
	}

	for namer := range layout.IterateNamers(fs) {
		if err := cache.ensure(namer.root, namer.slot, namer.index); err != nil {
			log.WithError(err).WithField("path", namer.path()).Error("Unable to cache blob metadata.")
		}
	}

	w.warmed = true
	close(w.ready)
	return nil
}

type cacheWarmer struct {
	mu     sync.Mutex
	warmed bool
	ready  chan struct{}
}
