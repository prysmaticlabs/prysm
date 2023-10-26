package das

import (
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

func newCache() *cache {
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
type dbidx [fieldparams.MaxBlobsPerBlock]*[48]byte

// missing compares the set of BlobSidecars observed in the backing store to the set of commitments
// observed in a block - cmts is the BlobKzgCommitments field from a block.
func (idx dbidx) missing(cmts [][]byte) ([]uint64, error) {
	m := make([]uint64, 0, len(cmts))
	for i := range cmts {
		if idx[i] == nil {
			m = append(m, uint64(i))
			continue
		}
		c := *idx[i]
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
	scs    [fieldparams.MaxBlobsPerBlock]*ethpb.BlobSidecar
	dbx    dbidx
	dbRead bool
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

func (e *cacheEntry) dbidxInitialized() bool {
	return e.dbRead
}

func (e *cacheEntry) cacheSlice(n int) ([]uint64, []*ethpb.BlobSidecar) {
	notInCache := make([]uint64, 0, n)
	for i := 0; i < len(e.scs); i++ {
		if e.scs[i] == nil {
			notInCache = append(notInCache, uint64(i))
		}
	}
	return notInCache, e.scs[0:n]
}

// ensureDbidx updates the db cache representation to include the given BlobSidecars.
func (e *cacheEntry) ensureDbidx(scs ...*ethpb.BlobSidecar) dbidx {
	if e.dbRead == false {
		e.dbRead = true
	}
	for i := range scs {
		if scs[i].Index >= fieldparams.MaxBlobsPerBlock {
			continue
		}
		// Don't overwrite.
		if e.dbx[scs[i].Index] != nil {
			continue
		}
		c := bytesutil.ToBytes48(scs[i].KzgCommitment)
		e.dbx[scs[i].Index] = &c
	}
	return e.dbx
}
