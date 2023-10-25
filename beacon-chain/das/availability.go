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
	errDBInconsistentWithBlock = errors.New("a value saved to the db is inconsistent with observed block commitments")
	errDAIncomplete            = errors.New("some commitments are not available at this time")
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
		if entry.persist(sc[i]) {
			persisted = append(persisted, sc[i])
		}
	}
	return persisted
}

// IsDataAvailable returns nil if all the commitments in the given block are persisted to the db and have been verified.
// - BlobSidecars already in the db are assumed to have been previously verified against the block.
// - BlobSidecars waiting for verification in the cache will be persisted to the db after verification.
// - When BlobSidecars are written to the db, their cache entries are cleared.
func (s *CachingDBVerifiedStore) IsDataAvailable(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error {
	c := commitmentsToCheck(b, current)
	if len(c) == 0 {
		// TODO: trigger cache cleanup?
		return nil
	}
	key := keyFromBlock(b)
	entry := s.cache.get(key)
	if entry == nil {
		// We only persist BlobSidecars after we have proven that their KzgCommitment verifies against their KzgProof, and
		// that the KzgCommitment matches the one at the corresponding index of the block. So if all of the block commitments
		// are present in the database, we have previously proven the data available and can reuse that result.
		dbidx, err := s.persisted(ctx, key)
		if err != nil {
			return err
		}
		missing, err := dbidx.missing(c)
		if err != nil {
			return err
		}
		if len(missing) > 0 {
			return NewMissingIndicesError(b.Root(), missing)
		}
		return nil
	}

	m, scs := entry.filterByBlock(c)
	// Happy cache path - we have all the sidecars we need in cache, and we just need to verify them.
	if m == len(c) {
		if err := kzg.IsDataAvailable(c, scs); err == nil {
			// We have all the committed sidecars in cache. If flushing them to succeeds, then we can confirm DA.
			if err := s.db.SaveBlobSidecar(ctx, scs); err != nil {
				return err
			}
			s.cache.deleteEntry(key)
			return nil
		}
	}

	// Since we don't have all the sidecars we need, we'll try the more complicated check:
	// merge the blobs in cache with those observed in the db. Verify the commitments one by one
	// and write any that match back to the db. Anything already written to the db can be assumed correct, as it was
	// previously observed in a block with the given root.
	return s.bisectPruneOrSave(ctx, key, c, scs)
}

func (s *CachingDBVerifiedStore) bisectPruneOrSave(ctx context.Context, key cacheKey, blockCommitments [][]byte, scs []*ethpb.BlobSidecar) error {
	dbidx, err := s.persisted(ctx, key)
	if err != nil {
		return err
	}
	missing, err := dbidx.missing(blockCommitments)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}

	bisect := make([]*ethpb.BlobSidecar, 0, len(missing))
	for i := range missing {
		idx := missing[i]
		if scs[idx] == nil {
			continue
		}
		bisect = append(bisect, scs[idx])
	}

	dbidx, err = s.persistIfProven(ctx, key, bisect)
	if err != nil {
		return err
	}

	missing, err = dbidx.missing(blockCommitments)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return NewMissingIndicesError(key.root, missing)
	}

	// no more missing - success!
	return nil
}

// persistIfProven saves all of the given BlobSidecars that pass proof verification. The BlobSidecars own stated commitment is used
// as the commitment to check against, so it is critical this method only be used where the KzgCommitment for a BlobSidecar has been
// matched against the commitment at the corresponding index in the block's set of commitments.
func (s *CachingDBVerifiedStore) persistIfProven(ctx context.Context, key cacheKey, scs []*ethpb.BlobSidecar) (dbidx, error) {
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
			log.WithField("block_root", fmt.Sprintf("%#x", key.root)).WithField("index", scs[i].Index).
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
	return s.cache.ensure(key).moveToDB(save...), nil
}

// persisted populate the db cache, which contains a mapping from Index->KzgCommitment for BlobSidecars previously verified
// (proof verification) and saved to the backend.
func (s *CachingDBVerifiedStore) persisted(ctx context.Context, key cacheKey) (dbidx, error) {
	entry := s.cache.ensure(key)
	// Cache is initialized with a nil value to differentiate absent BlobSidecars from an uninitialized db cache.
	if entry.dbidx != nil {
		return entry.dbidx, nil
	}
	sidecars, err := s.db.BlobSidecarsByRoot(ctx, key.root)
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
