package das

import (
	"context"

	errors "github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/kzg"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
)

var errDAIncomplete = errors.New("some commitments are not available at this time")

// BlobsDB specifies the persistence store methods needed by the AvailabilityStore.
type BlobsDB interface {
	BlobSidecarsByRoot(ctx context.Context, beaconBlockRoot [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error)
	SaveBlobSidecar(ctx context.Context, sidecars []*ethpb.BlobSidecar) error
}

// AvailabilityStore describes a component that can verify and save sidecars for a given block, and confirm previously
// verified and saved sidecars.
type AvailabilityStore interface {
	IsDataAvailable(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error
	PersistOnceCommitted(ctx context.Context, current primitives.Slot, sc ...*ethpb.BlobSidecar) []*ethpb.BlobSidecar
}

type CachingDBVerifiedStore struct {
	db    BlobsDB
	cache *cache
}

var _ AvailabilityStore = &CachingDBVerifiedStore{}

func NewCachingDBVerifiedStore(db BlobsDB) *CachingDBVerifiedStore {
	return &CachingDBVerifiedStore{
		db:    db,
		cache: newCache(),
	}
}

// PersistOnceCommitted adds blobs to the working blob cache (in-memory or disk backed is an implementation
// detail). Blobs stored in this cache will be persisted for at least as long as the node is
// running. Once IsDataAvailable succeeds, all blobs referenced by the given block are guaranteed
// to be persisted for the remainder of the retention period.
func (s *CachingDBVerifiedStore) PersistOnceCommitted(ctx context.Context, current primitives.Slot, sc ...*ethpb.BlobSidecar) []*ethpb.BlobSidecar {
	var key cacheKey
	var entry *cacheEntry
	persisted := make([]*ethpb.BlobSidecar, 0, len(sc))
	for i := range sc {
		if !params.WithinDAPeriod(slots.ToEpoch(sc[i].Slot), slots.ToEpoch(current)) {
			continue
		}
		if sc[i].Index > fieldparams.MaxBlobsPerBlock-1 {
			log.WithField("block_root", sc[i].BlockRoot).WithField("index", sc[i].Index).Error("discarding BlobSidecar with out of bound commitment")
		}
		skey := keyFromSidecar(sc[i])
		if key != skey {
			key = skey
			entry = s.cache.ensure(key)
		}
		if entry.stash(sc[i]) {
			persisted = append(persisted, sc[i])
		}
	}
	return persisted
}

// IsDataAvailable returns nil if all the commitments in the given block are persisted to the db and have been verified.
// - BlobSidecars already in the db are assumed to have been previously verified against the block.
// - BlobSidecars waiting for verification in the cache will be persisted to the db after verification.
// - When BlobSidecars are written to the db, their cache entries are cleared.
// - BlobSidecar cache entries with commitments that do not match the block will be evicted.
// - BlobSidecar cachee entries with commitments that fail proof verification will be evicted.
func (s *CachingDBVerifiedStore) IsDataAvailable(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error {
	blockCommitments := commitmentsToCheck(b, current)
	if len(blockCommitments) == 0 {
		return nil
	}

	key := keyFromBlock(b)
	entry := s.cache.ensure(key)
	// holding the lock over the course of the DA check simplifies everything
	entry.Lock()
	defer entry.Unlock()
	if err := s.daCheck(ctx, b.Root(), blockCommitments, entry); err != nil {
		return err
	}
	// If there is no error, DA has been successful, so we can clean up the cache.
	s.cache.delete(key)
	return nil
}

func (s *CachingDBVerifiedStore) daCheck(ctx context.Context, root [32]byte, blockCommitments [][]byte, entry *cacheEntry) error {
	notInCache, sidecars := entry.cacheSlice(len(blockCommitments))
	if len(notInCache) == 0 {
		if err := kzg.IsDataAvailable(blockCommitments, sidecars); err == nil {
			// We have all the committed sidecars in cache, and they all have valid proofs.
			// If flushing them to backing storage succeeds, then we can confirm DA.
			return s.db.SaveBlobSidecar(ctx, sidecars)
		}
	}
	// Check if we have the commitments already in the database.
	dbidx, err := s.persisted(ctx, root, entry)
	// persisted() accounts for db.ErrNotFound, so this is a real database error.
	if err != nil {
		return err
	}
	notInDb, err := dbidx.missing(blockCommitments)
	// This is a database integrity sanity check - it should never fail.
	if err != nil {
		return err
	}
	// All commitments were found in the db, due to a previous successful DA check.
	if len(notInDb) == 0 {
		return nil
	}
	// Return an error that indicates which committed indices are not in the cache.
	return NewMissingIndicesError(notInCache)
}

// persisted populate the db cache, which contains a mapping from Index->KzgCommitment for BlobSidecars previously verified
// (proof verification) and saved to the backend.
func (s *CachingDBVerifiedStore) persisted(ctx context.Context, root [32]byte, entry *cacheEntry) (dbidx, error) {
	if entry.dbidxInitialized() {
		return entry.dbidx(), nil
	}
	sidecars, err := s.db.BlobSidecarsByRoot(ctx, root)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// No BlobSidecars, initialize with empty idx.
			return entry.ensureDbidx(), nil
		}
		return entry.dbidx(), err
	}
	// Ensure all sidecars found in the db are represented in the cache and return the cache value.
	return entry.ensureDbidx(sidecars...), nil
}

func commitmentsToCheck(b blocks.ROBlock, current primitives.Slot) [][]byte {
	if b.Version() < version.Deneb {
		return nil
	}
	// We are only required to check within MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS
	if !params.WithinDAPeriod(slots.ToEpoch(b.Block().Slot()), slots.ToEpoch(current)) {
		return nil
	}
	kzgCommitments, err := b.Block().Body().BlobKzgCommitments()
	if err != nil {
		return nil
	}
	return kzgCommitments
}
