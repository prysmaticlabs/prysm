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
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
)

var (
	errDBInconsistentWithBlock = errors.New("a value saved to the db is inconsistent with observed block commitments")
	errDAIncomplete            = errors.New("some commitments are missing at this time")
)

// BlobsDB specifies the persistence store methods needed by the AvailabilityStore.
type BlobsDB interface {
	BlobSidecarsByRoot(ctx context.Context, beaconBlockRoot [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error)
	SaveBlobSidecar(ctx context.Context, sidecars []*ethpb.BlobSidecar) error
}

// AvailabilityStore describes a component that can verify and save sidecars for a given block, and confirm previously
// verified and saved sidecars.
type AvailabilityStore interface {
	VerifyAvailability(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error
	PersistBlobs(ctx context.Context, current primitives.Slot, sc ...*ethpb.BlobSidecar)
	//WaitToVerify(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error
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

// PersistBlobs adds blobs to the working blob cache (in-memory or disk backed is an implementation
// detail). Blobs stored in this cache will be persisted for at least as long as the node is
// running. Once VerifyAvailability succeed, all blobs referenced by the given block are guaranteed
// to be persisted for the remainder of the retention period.
func (s *CachingDBVerifiedStore) PersistBlobs(ctx context.Context, current primitives.Slot, sc ...*ethpb.BlobSidecar) {
	if len(sc) < 1 {
		return
	}
	var key cacheKey
	var entry *cacheEntry
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
		entry.persist(sc[i])
	}
}

func (s *CachingDBVerifiedStore) VerifyAvailability(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error {
	c := commitmentsToCheck(b, current)
	if len(c) == 0 {
		// TODO: trigger cache cleanup?
		return nil
	}
	key := keyFromBlock(b)
	entry := s.cache.get(key)
	if entry == nil {
		return s.databaseDACheck(ctx, current, b.Root(), c)
	}
	m, scs := entry.filterByBlock(c)
	// Happy path - we have all the sidecars we need in cache, and we just need to verify them.
	if m == len(c) {
		if err := kzg.IsDataAvailable(c, scs); err == nil {
			// We have all the committed sidecars in cache. If flushing them to succeeds, then we can confirm DA.
			if err := s.db.SaveBlobSidecar(ctx, scs); err != nil {
				return err
			}
			s.cache.delete(keyFromSidecar(scs[0]))
		}
	}
	// Since we don't have all the sidecars we need, we'll try the more complicated check:
	// merge the blobs in cache with those observed in the db. Verify the commitments one by one
	// and write any that match back to the db. Anything already written to the db can be assumed correct, as it was
	// previously observed in a block with the given root.
	return s.bisectPruneOrSave(ctx, b.Root(), c, scs)
}

func (s *CachingDBVerifiedStore) bisectPruneOrSave(ctx context.Context, root [32]byte, cmts [][]byte, scs []*ethpb.BlobSidecar) error {
	dbscs, err := s.db.BlobSidecarsByRoot(ctx, root)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		log.WithError(err).Error("failed to lookup saved BlobSidecars")
	}

	check := make([]*ethpb.BlobSidecar, len(cmts))
	for i := range dbscs {
		if dbscs[i].Index >= uint64(len(cmts)) {
			// TODO: This check is necessary to avoid a panic but shouldn't be possible, because it would mean we've seen 2 versions of a block
			// that somehow have the same root, but different commitments.
			return errors.Wrapf(errDBInconsistentWithBlock, "db BlobSidecar.Index=%d > len(block.KzgCommitments)=%d", dbscs[i].Index, len(cmts))
		}
		check[dbscs[i].Index] = dbscs[i]
	}

	bisect := make([]*ethpb.BlobSidecar, 0, len(cmts))
	incomplete := false
	for i := range check {
		if check[i] != nil {
			continue
		}
		if i < len(scs) && scs[i] != nil {
			bisect = append(bisect, scs[i])
			check[i] = scs[i]
			continue
		}
		incomplete = true
	}

	var daErr error
	if !incomplete {
		// We don't want to react to this error yet, because there could still be good sidecars to save.
		// So stash the error in daErr to return later.
		if daErr = kzg.IsDataAvailable(cmts, check); daErr == nil {
			if len(bisect) > 0 {
				if err := s.db.SaveBlobSidecar(ctx, bisect); err != nil {
					return errors.Wrap(err, "unable to save BlobSidecars to complete DA check")
				}
			}
			s.cache.delete(keyFromSidecar(check[0]))
			return nil
		}
	}

	// Either we have an incomplete set of commitments, or we failed the da test.
	// In either case, check if any of the sidecars from the cache are valid. If so save them to the db.
	save := make([]*ethpb.BlobSidecar, 0, len(bisect))
	indices := make([]uint64, 0, len(bisect))
	for i := range bisect {
		if err := kzg.IsDataAvailable([][]byte{bisect[i].KzgCommitment}, []*ethpb.BlobSidecar{bisect[i]}); err != nil {
			log.WithField("block_root", fmt.Sprintf("%#x", root)).WithField("index", bisect[i].Index).
				WithField("commitment", fmt.Sprintf("%#x", bisect[i].KzgCommitment)).WithError(err).
				Error("commitment proof failure")
			continue
		}
		// DA check was successful for this BlobSidecar, so we'll save it to the db.
		save = append(save, bisect[i])
		indices = append(indices, bisect[i].Index)
	}
	if len(save) > 0 {
		if err := s.db.SaveBlobSidecar(ctx, save); err != nil {
			log.WithError(err).Error("failed to save BlobSidecar")
		}
		s.cache.delete(keyFromSidecar(save[0]), indices...)
	}

	if daErr != nil {
		return daErr
	}
	return errDAIncomplete
}

func (s *CachingDBVerifiedStore) databaseDACheck(ctx context.Context, current primitives.Slot, root [32]byte, cmts [][]byte) error {
	log.WithField("root", fmt.Sprintf("%#x", root)).Info("Falling back to database DA check")
	sidecars, err := s.db.BlobSidecarsByRoot(ctx, root)
	if err != nil {
		return errors.Wrap(err, "could not get blob sidecars")
	}
	if err := kzg.IsDataAvailable(cmts, sidecars); err != nil {
		return err
	}
	return nil
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
