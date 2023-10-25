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

type dbidx map[uint64][48]byte

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

type cacheEntry struct {
	sync.RWMutex
	scs   [fieldparams.MaxBlobsPerBlock]*ethpb.BlobSidecar
	dbidx dbidx
}

func (e *cacheEntry) ensureDbidx(scs ...*ethpb.BlobSidecar) dbidx {
	e.Lock()
	defer e.Unlock()
	if e.dbidx == nil {
		e.dbidx = make(dbidx)
	}
	for i := range scs {
		e.dbidx[scs[i].Index] = bytesutil.ToBytes48(scs[i].KzgCommitment)
	}
	return e.dbidx
}

func (e *cacheEntry) persist(sc *ethpb.BlobSidecar) bool {
	e.Lock()
	defer e.Unlock()
	if e.scs[sc.Index] == nil {
		e.scs[sc.Index] = sc
		return true
	}
	return false
}

// filterByBlock evicts any BlobSidecars that do not have a KzgCommitment field that matches the
// corresponding index in the block's BlobKzgCommitments field.
// The first return value is the number of commitments that matched the block, and the second is a
// slice of the BlobSidecars that are left after filtering out mismatches.
func (e *cacheEntry) filterByBlock(blkCmts [][]byte) (int, []*ethpb.BlobSidecar) {
	matches := 0
	scs := make([]*ethpb.BlobSidecar, len(blkCmts))
	e.Lock()
	defer e.Unlock()
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

func (e *cacheEntry) get(idx uint64) *ethpb.BlobSidecar {
	e.RLock()
	defer e.RUnlock()
	return e.scs[idx]
}

// delete removes all the requested indices from the cache. If
func (e *cacheEntry) delete(idx ...uint64) {
	e.Lock()
	defer e.Unlock()
	for i := range idx {
		e.scs[idx[i]] = nil
	}
}

func (e *cacheEntry) lockFreeCount() int {
	nn := 0
	for i := range e.scs {
		if e.scs[i] != nil {
			nn += 1
		}
	}
	return nn
}

func (e *cacheEntry) moveToDB(scs ...*ethpb.BlobSidecar) dbidx {
	e.Lock()
	defer e.Unlock()
	for i := range scs {
		sc := scs[i]
		e.dbidx[sc.Index] = bytesutil.ToBytes48(sc.KzgCommitment)
		e.scs[sc.Index] = nil
	}
	return e.dbidx
}

func newCacheEntry() *cacheEntry {
	// dbidx is initialized with a nil value to differentiate absent BlobSidecars from an uninitialized db cache.
	return &cacheEntry{dbidx: nil}
}

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

func keyFromSidecar(sc *ethpb.BlobSidecar) cacheKey {
	return cacheKey{slot: sc.Slot, root: bytesutil.ToBytes32(sc.BlockRoot)}
}

func keyFromBlock(b blocks.ROBlock) cacheKey {
	return cacheKey{slot: b.Block().Slot(), root: b.Root()}
}

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

func (c *cache) get(key cacheKey) *cacheEntry {
	c.RLock()
	defer c.RUnlock()
	return c.entries[key]
}

func (c *cache) deleteEntry(key cacheKey) {
	c.Lock()
	defer c.Unlock()
	delete(c.entries, key)
}

func (c *cache) delete(key cacheKey, idx ...uint64) {
	c.Lock()
	defer c.Unlock()
	entry := c.entries[key]
	if entry != nil {
		entry.delete(idx...)
	}
	entry.Lock()
	defer entry.Unlock()
	if entry.lockFreeCount() == 0 {
		delete(c.entries, key)
	}
}
