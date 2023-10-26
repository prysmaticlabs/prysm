package das

import (
	"bytes"
	"sync"

	errors "github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

var (
	errDBCommitmentMismatch = errors.New("blob/block commitment mismatch")
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

func newSidecarCache() *cache {
	return &cache{entries: make(map[cacheKey]*cacheEntry)}
}

// keyFromSidecar is a convenience method for constructing a cacheKey from a BlobSidecar value.
func keyFromSidecar(sc *ethpb.BlobSidecar) cacheKey {
	return cacheKey{slot: sc.Slot, root: bytesutil.ToBytes32(sc.BlockRoot)}
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
		e = newCacheEntry()
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
type dbidx map[uint64][48]byte

// missing compares the set of BlobSidecars observed in the backing store to the set of commitments
// observed in a block - cmts is the BlobKzgCommitments field from a block.
func (idx dbidx) missing(cmts [][]byte) ([]uint64, error) {
	m := make([]uint64, 0, len(cmts))
	for i := range cmts {
		x := uint64(i)
		c, ok := idx[x]
		if !ok {
			m = append(m, x)
			continue
		}
		if c != bytesutil.ToBytes48(cmts[i]) {
			return nil, errors.Wrapf(errDBCommitmentMismatch,
				"index=%d, db=%#x, block=%#x", i, c, cmts[i])
		}
	}
	return m, nil
}

// cacheEntry represents 2 different types of caches for a given block.
// scs is a fixed-length cache of BlobSidecars.
// dbx is a compact representation of BlobSidecars observed in the backing store.
// dbx assumes that all writes to the backing store go through the same cache.
type cacheEntry struct {
	sync.RWMutex
	scs [fieldparams.MaxBlobsPerBlock]*ethpb.BlobSidecar
	dbx dbidx
}

// stash adds an item to the in-memory cache of BlobSidecars.
// Only the first BlobSidecar of a given Index will be kept in the cache.
// The return value represents whether the given BlobSidecar was stashed.
// A false value means there was already a BlobSidecar with the given Index.
func (e *cacheEntry) stash(sc *ethpb.BlobSidecar) bool {
	if e.scs[sc.Index] == nil {
		e.scs[sc.Index] = sc
		return true
	}
	return false
}

func (e *cacheEntry) dbidx() dbidx {
	return e.dbx
}

// filter evicts any BlobSidecars that do not have a KzgCommitment field that matches blkCmts,
// which is the block's BlobKzgCommitments field.
// The first return value is the number of commitments that matched the block, ie the number of
// non-nil vaues in the second return value, which is a copy of the slice of cached BlobSidecars
// that match blkCmts.
func (e *cacheEntry) filter(blkCmts [][]byte) (int, []*ethpb.BlobSidecar) {
	matches := 0
	scs := make([]*ethpb.BlobSidecar, len(blkCmts))
	for i := range blkCmts {
		// Clear any blobs that don't match the block.
		if !bytes.Equal(blkCmts[i], e.scs[i].KzgCommitment) {
			e.scs[i] = nil
		} else {
			matches += 1
		}
	}
	// Clear any blobs that the block doesn't include.
	for i := len(blkCmts); i < fieldparams.MaxBlobsPerBlock; i++ {
		e.scs[i] = nil
	}
	copy(scs, e.scs[:])
	return matches, scs
}

// moveToDB evicts then given BlobSidecars from the in-memory cache, and updates the
// db cache representation to include them.
func (e *cacheEntry) moveToDB(scs ...*ethpb.BlobSidecar) dbidx {
	e.ensureDbidx(scs...)
	for i := range scs {
		e.scs[scs[i].Index] = nil
	}
	return e.dbx
}

// ensureDbidx updates the db cache representation to include the given BlobSidecars.
func (e *cacheEntry) ensureDbidx(scs ...*ethpb.BlobSidecar) dbidx {
	if e.dbx == nil {
		e.dbx = make(dbidx)
	}
	for i := range scs {
		e.dbx[scs[i].Index] = bytesutil.ToBytes48(scs[i].KzgCommitment)
	}
	return e.dbx
}

func newCacheEntry() *cacheEntry {
	// dbidx is initialized with a nil value to differentiate absent BlobSidecars from an uninitialized db cache.
	return &cacheEntry{dbx: nil}
}
