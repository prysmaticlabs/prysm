package filesystem

import (
	"sync"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
)

const bytesPerSidecar = 131928

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
	mu     sync.RWMutex
	nBlobs float64
	cache  map[[32]byte]BlobStorageSummary
}

var _ BlobStorageSummarizer = &blobStorageCache{}

func newBlobStorageCache() *blobStorageCache {
	return &blobStorageCache{
		cache: make(map[[32]byte]BlobStorageSummary),
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

func (s *blobStorageCache) get(key [32]byte) (BlobStorageSummary, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.cache[key]
	return v, ok
}

func (s *blobStorageCache) identForIdx(key [32]byte, idx uint64) (blobIdent, error) {
	v, ok := s.get(key)
	if !ok || !v.HasIndex(idx) {
		return blobIdent{}, db.ErrNotFound
	}
	return blobIdent{
		root:  key,
		index: idx,
		epoch: v.epoch,
	}, nil
}

func (s *blobStorageCache) identForRoot(key [32]byte) (blobIdent, error) {
	v, ok := s.get(key)
	if !ok {
		return blobIdent{}, db.ErrNotFound
	}
	return blobIdent{
		root:  key,
		epoch: v.epoch,
	}, nil
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
