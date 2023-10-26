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
	scs [fieldparams.MaxBlobsPerBlock]*ethpb.BlobSidecar
	dbx dbidx
}

func (e *cacheEntry) persist(sc *ethpb.BlobSidecar) bool {
	if e.scs[sc.Index] == nil {
		e.scs[sc.Index] = sc
		return true
	}
	return false
}

func (e *cacheEntry) dbidx() dbidx {
	return e.dbx
}

// filter evicts any BlobSidecars that do not have a KzgCommitment field that matches the
// corresponding index in the block's BlobKzgCommitments field.
// The first return value is the number of commitments that matched the block, and the second is a
// slice of the BlobSidecars that are left after filtering out mismatches.
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

// delete removes all the requested indices from the cache. If
func (e *cacheEntry) delete(idx ...uint64) {
	for i := range idx {
		e.scs[idx[i]] = nil
	}
}

func (e *cacheEntry) moveToDB(scs ...*ethpb.BlobSidecar) dbidx {
	for i := range scs {
		sc := scs[i]
		e.dbx[sc.Index] = bytesutil.ToBytes48(sc.KzgCommitment)
		e.scs[sc.Index] = nil
	}
	return e.dbx
}

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

func (c *cache) delete(key cacheKey) {
	c.Lock()
	defer c.Unlock()
	delete(c.entries, key)
}
