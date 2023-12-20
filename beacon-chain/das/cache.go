package das

import (
	"bytes"

	errors "github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
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
	e, ok := c.entries[key]
	if !ok {
		e = &cacheEntry{}
		c.entries[key] = e
	}
	return e
}

// delete removes the cache entry from the cache.
func (c *cache) delete(key cacheKey) {
	delete(c.entries, key)
}

// cacheEntry holds a fixed-length cache of BlobSidecars.
type cacheEntry struct {
	scs [fieldparams.MaxBlobsPerBlock]*blocks.ROBlob
}

// stash adds an item to the in-memory cache of BlobSidecars.
// Only the first BlobSidecar of a given Index will be kept in the cache.
// stash will return an error if the given blob is already in the cache, or if the Index is out of bounds.
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

/*
// filter returns a slice of all sidecars with KzgCommitment values that are found
// in the given list of block commitments, ignoring any that shouldn't be there.
// This is a cheap pre-filtering step before full blob verification.
func (e *cacheEntry) filter(root [32]byte, blkCmts [][]byte) []blocks.ROBlob {
	upper := fieldparams.MaxBlobsPerBlock
	if upper > len(blkCmts) {
		upper = len(blkCmts)
	}
	want := make([]blocks.ROBlob, 0, len(e.scs))
	for i := 0; i < upper; i++ {
		if e.scs[i] == nil {
			continue
		}
		if bytes.Equal(blkCmts[i], e.scs[i].KzgCommitment) {
			want = append(want, *e.scs[i])
		}
	}
	return want
}
*/

var (
	errCommitmentMismatch = errors.New("KzgCommitment of sidecar in cache did not match block commitment")
	errMissingSidecar     = errors.New("no sidecar in cache for block commitment")
)

// filter evicts sidecars that are not committed to by the block and returns custom
// errors if the cache is missing any of the commitments, or if the commitments in
// the cache do not match those found in the block. If err is nil, then all expected
// commitments were found in the cache and the sidecar slice return value can be used
// to perform a DA check against the cached sidecars.
func (e *cacheEntry) filter(root [32]byte, kc safeCommitmentArray) ([]blocks.ROBlob, error) {
	scs := make([]blocks.ROBlob, kc.count())
	for i := uint64(0); i < fieldparams.MaxBlobsPerBlock; i++ {
		if kc[i] == nil {
			if e.scs[i] != nil {
				return nil, errors.Wrapf(errCommitmentMismatch, "root=%#x, index=%#x, commitment=%#x, no block commitment", root, i, e.scs[i].KzgCommitment)
			}
			continue
		}

		if e.scs[i] == nil {
			return nil, errors.Wrapf(errMissingSidecar, "root=%#x, index=%#x", root, i)
		}
		if !bytes.Equal(kc[i], e.scs[i].KzgCommitment) {
			return nil, errors.Wrapf(errCommitmentMismatch, "root=%#x, index=%#x, commitment=%#x, block commitment=%#x", root, i, e.scs[i].KzgCommitment, kc[i])
		}
		scs[i] = *e.scs[i]
	}

	return scs, nil
}

// dbidx is a compact representation of the set of BlobSidecars in the database for a given root,
// organized as a map from BlobSidecar.Index->BlobSidecar.KzgCommitment.
// This representation is convenient for comparison to a block's commitments.
type dbidx [fieldparams.MaxBlobsPerBlock]bool

// missing compares the set of BlobSidecars observed in the backing store to the set of commitments
// observed in a block - cmts is the BlobKzgCommitments field from a block.
func (idx dbidx) missing(expected int) []uint64 {
	if expected > fieldparams.MaxBlobsPerBlock {
		expected = fieldparams.MaxBlobsPerBlock
	}
	m := make([]uint64, 0, expected)
	for i := 0; i < expected; i++ {
		if !idx[i] {
			m = append(m, uint64(i))
			continue
		}
	}
	return m
}

// safeCommitemntArray is a fixed size array of commitment byte slices. This is helpful for avoiding
// gratuitous bounds checks.
type safeCommitmentArray [fieldparams.MaxBlobsPerBlock][]byte

func (s safeCommitmentArray) count() int {
	for i := range s {
		if s[i] == nil {
			return i
		}
	}
	return fieldparams.MaxBlobsPerBlock
}
