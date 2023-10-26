package das

import (
	"context"
	"fmt"

	errors "github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/kzg"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
)

var (
	errDAIncomplete = errors.New("some commitments are not available at this time")
)

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
		cache: newSidecarCache(),
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
	// retrieve BlobSidecars from the cache that match the block commitments. The order of blockCommitments is significant -
	// the array offset of each [48]byte is assumed to correspond to the .Index field of the related BlobSidecar.
	m, cachedScs := entry.filter(blockCommitments)
	// The first return value from filter() is the number of items in the cache that match their corresponding commitments.
	// If all BlobSidecars for the block are in the cache, we can avoid touching the database as a special case.
	// This is usually the case when the cache is used for initial-sync or backfill, where caches can be ephemeral (one per batch).
	if m == len(blockCommitments) {
		if err := kzg.IsDataAvailable(blockCommitments, cachedScs); err == nil {
			// We have all the committed sidecars in cache, and they all have valid proofs.
			// If flushing them to backing storage succeeds, then we can confirm DA.
			return s.db.SaveBlobSidecar(ctx, cachedScs)
		}
	}

	// Since we don't have all the sidecars we need in cache, we'll try the more complicated check:
	// merge the blobs in cache with those observed in the db. Try to verify all commitments together, and failing that,
	// verify them one-by-one to find the set that match, and write the proven committed BlobSidecars back to the db.
	// Anything already written to the db can be assumed correct - previously a block with the given root was observed
	// that committed to those blobs, and their commitments to the data were proven.
	dbidx, err := s.persisted(ctx, root, entry)
	if err != nil {
		return err
	}
	missing, err := dbidx.missing(blockCommitments)
	// This is a database integrity sanity check - it should never fail.
	if err != nil {
		return err
	}
	// All commitments were found in the db, due to a previous successful DA check.
	if len(missing) == 0 {
		return nil
	}
	// Some commitments are missing, and we know they aren't in the cache because there are no matching commitments in the cache.
	// This means DA check fails (at this time).
	if m == 0 {
		return NewMissingIndicesError(missing)
	}

	// If we have any of the missing commitments in the cache, collect them to be verified and persisted.
	bisect := make([]*ethpb.BlobSidecar, 0, len(missing))
	for i := range missing {
		idx := missing[i]
		if cachedScs[idx] == nil {
			continue
		}
		bisect = append(bisect, cachedScs[idx])
	}

	// persistIfProven returns the updated value of dbidx, which we can use to check if any comitments are still missing.
	dbidx, err = s.persistIfProven(ctx, entry, bisect)
	if err != nil {
		return err
	}

	missing, err = dbidx.missing(blockCommitments)
	if err != nil {
		return err
	}
	// If there are still missing commitments, DA check fails (for now).
	if len(missing) > 0 {
		return NewMissingIndicesError(missing)
	}

	// no more missing - success!
	return nil
}

// persistIfProven saves all of the given BlobSidecars that pass proof verification. The BlobSidecars own stated commitment is used
// as the commitment to check against, so it is critical this method only be used where the KzgCommitment for a BlobSidecar has been
// matched against the commitment at the corresponding index in the block's set of commitments.
func (s *CachingDBVerifiedStore) persistIfProven(ctx context.Context, entry *cacheEntry, scs []*ethpb.BlobSidecar) (dbidx, error) {
	failed := make(map[[48]byte]struct{})

	err := kzg.BisectBlobSidecarKzgProofs(scs)
	if err != nil {
		var errFailed *kzg.KzgProofError
		// Can't cast to KzgProofError to see which commitments failed
		if !errors.As(err, errFailed) {
			return nil, err
		}

		// The custom error returned by BisectBlobSidecarKzgProofs provides the Failed method to retrieve a list of commitments
		// that failed verification after bisection. Use this to filter out BlobSidecars and only save the ones that were successfully
		// proven.
		for _, fc := range errFailed.Failed() {
			failed[fc] = struct{}{}
		}
	}

	save := make([]*ethpb.BlobSidecar, 0, len(scs)-len(failed))
	for i := range scs {
		car := bytesutil.ToBytes48(scs[i].KzgCommitment)
		// Skipping failed commitments.
		if _, ok := failed[car]; ok {
			log.WithField("block_root", fmt.Sprintf("%#x", scs[i].BlockRoot)).WithField("index", scs[i].Index).
				WithField("commitment", fmt.Sprintf("%#x", scs[i])).WithError(err).
				Error("commitment proof failure. evicting BlobSidecar from cache and not saving to db")
			continue
		}
		// All other BlobSidecars were successfully proven, so they should be saved to the db.
		save = append(save, scs[i])
	}
	// Save any of the Sidecars where the commitments passed proof verification.
	if err := s.db.SaveBlobSidecar(ctx, save); err != nil {
		// If any of them fail, there is something critically wrong with the db, and we can't give any availability guarantees,
		// so we need to bubble this error.
		return nil, err
	}
	return entry.moveToDB(save...), nil
}

// persisted populate the db cache, which contains a mapping from Index->KzgCommitment for BlobSidecars previously verified
// (proof verification) and saved to the backend.
func (s *CachingDBVerifiedStore) persisted(ctx context.Context, root [32]byte, entry *cacheEntry) (dbidx, error) {
	dbidx := entry.dbidx()
	// Cache is initialized with a nil value to differentiate absent BlobSidecars from an uninitialized db cache.
	// A non-nil value means that persisted() has run already, so we can trust the value is current.
	if dbidx != nil {
		return dbidx, nil
	}
	sidecars, err := s.db.BlobSidecarsByRoot(ctx, root)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// No BlobSidecars, initialize with empty idx.
			return entry.ensureDbidx(), nil
		}
		return nil, err
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
