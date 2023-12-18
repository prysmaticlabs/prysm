package das

import (
	"bytes"
	"fmt"
	"sync"

	errors "github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	log "github.com/sirupsen/logrus"
)

var (
	ErrDuplicateSidecar = errors.New("duplicate sidecar stashed in AvailabilityStore")
	errIndexOutOfBounds = errors.New("sidecar.index > MAX_BLOBS_PER_BLOCK")
)

// cacheKey includes the slot so that we can easily iterate through the cache and compare
// slots for eviction purposes. Whether the input is the block or the sidecar, we always have
// the root+slot when interacting with the cache, so it isn't an inconvenience to use both.
type cacheKey struct {
	slot primitives.Slot
	root [32]byte
}

type cache struct {
	sync.RWMutex
	entries map[cacheKey]*cacheEntry
}

func newCache() *cache {
	return &cache{entries: make(map[cacheKey]*cacheEntry)}
}

// keyFromSidecar is a convenience method for constructing a cacheKey from a BlobSidecar value.
func keyFromSidecar(sc blocks.ROBlob) cacheKey {
	return cacheKey{slot: header(sc).Slot, root: sc.BlockRoot()}
}

// keyFromBlock is a convenience method for constructing a cacheKey from a ROBlock value.
func keyFromBlock(b blocks.ROBlock) cacheKey {
	return cacheKey{slot: b.Block().Slot(), root: b.Root()}
}

// ensure returns the entry for the given key, creating it if it isn't already present.
func (c *cache) ensure(key cacheKey) *cacheEntry {
	c.Lock()
	defer c.Unlock()
	e, ok := c.entries[key]
	if !ok {
		e = &cacheEntry{}
		c.entries[key] = e
	}
	return e
}

// delete removes the cache entry from the cache.
func (c *cache) delete(key cacheKey) {
	c.Lock()
	defer c.Unlock()
	delete(c.entries, key)
}

// dbidx is a compact representation of the set of BlobSidecars in the database for a given root,
// organized as a map from BlobSidecar.Index->BlobSidecar.KzgCommitment.
// This representation is convenient for comparison to a block's commitments.
type dbidx [fieldparams.MaxBlobsPerBlock]bool

// missing compares the set of BlobSidecars observed in the backing store to the set of commitments
// observed in a block - cmts is the BlobKzgCommitments field from a block.
func (idx dbidx) missing(expected int) []uint64 {
	m := make([]uint64, 0, expected)
	for i := 0; i < expected; i++ {
		if !idx[i] {
			m = append(m, uint64(i))
			continue
		}
	}
	return m
}

// cacheEntry represents 2 different types of caches for a given block.
// scs is a fixed-length cache of BlobSidecars.
// dbx is a compact representation of BlobSidecars observed in the backing store.
// dbx assumes that all writes to the backing store go through the same cache.
type cacheEntry struct {
	sync.RWMutex
	scs    [fieldparams.MaxBlobsPerBlock]*blocks.ROBlob
	dbx    dbidx
	dbRead bool
}

// stash adds an item to the in-memory cache of BlobSidecars.
// Only the first BlobSidecar of a given Index will be kept in the cache.
// The return value represents whether the given BlobSidecar was stashed.
// A false value means there was already a BlobSidecar with the given Index.
func (e *cacheEntry) stash(sc *blocks.ROBlob) error {
	if sc.Index >= fieldparams.MaxBlobsPerBlock {
		return errors.Wrapf(errIndexOutOfBounds, "index=%d", sc.Index)
	}
	if e.scs[sc.Index] != nil {
		return errors.Wrapf(ErrDuplicateSidecar, "root=%#x, index=%d, commitment=%#x", sc.BlockRoot(), sc.Index, sc.KzgCommitment)
	}
	e.scs[sc.Index] = sc
	return nil
}

func (e *cacheEntry) dbidx() dbidx {
	return e.dbx
}

func (e *cacheEntry) dbidxInitialized() bool {
	return e.dbRead
}

// filter evicts sidecars that are not committed to by the block and returns custom
// errors if the cache is missing any of the commitments, or if the commitments in
// the cache do not match those found in the block. If err is nil, then all expected
// commitments were found in the cache and the sidecar slice return value can be used
// to perform a DA check against the cached sidecars.
func (e *cacheEntry) filter(root [32]byte, blkCmts [][]byte) ([]blocks.ROBlob, error) {
	scs := make([]blocks.ROBlob, len(blkCmts))
	// Evict any blobs that are out of range.
	for i := len(blkCmts); i < fieldparams.MaxBlobsPerBlock; i++ {
		if e.scs[i] == nil {
			continue
		}
		log.WithField("block_root", root).
		  WithField("block_commitments", len(blkCmts)).
			WithField("sidecar_index", i).
			WithField("cached_commitment", fmt.Sprintf("%#x", e.scs[i].KzgCommitment)).
			Warn("Evicting BlobSidecar without commitment in block")
		e.scs[i] = nil
	}
	// Generate a MissingIndicesError for any missing indices.
	// Generate a CommitmentMismatchError for any mismatched commitments.
	missing := make([]uint64, 0, len(blkCmts))
	mismatch := make([]uint64, 0, len(blkCmts))
	for i := range blkCmts {
		if e.scs[i] == nil {
			missing = append(missing, uint64(i))
			continue
		}
		if !bytes.Equal(blkCmts[i], e.scs[i].KzgCommitment) {
			mismatch = append(mismatch, uint64(i))
			log.WithField("block_root", root).
				WithField("index", i).
				WithField("expected_commitment", fmt.Sprintf("%#x", blkCmts[i])).
				WithField("cached_commitment", fmt.Sprintf("%#x", e.scs[i].KzgCommitment)).
				Error("Evicting BlobSidecar with incorrect commitment")
			e.scs[i] = nil
			continue
		}
		scs[i] = *e.scs[i]
	}
	if len(mismatch) > 0 {
		return nil, NewCommitmentMismatchError(mismatch)
	}
	if len(missing) > 0 {
		return nil, NewMissingIndicesError(missing)
	}
	return scs, nil
}

// ensureDbidx updates the db cache representation to include the given BlobSidecars.
func (e *cacheEntry) ensureDbidx(fromDB [fieldparams.MaxBlobsPerBlock]bool) dbidx {
	e.dbRead = true
	for i := range fromDB {
		// Positive result in cache overrides negative result in db.
		if !e.dbx[i] && fromDB[i] {
			e.dbx[i] = fromDB[i]
		}
	}
	return e.dbx
}
