package das

import (
	"context"

	errors "github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filesystem"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

var (
	errMixedRoots = errors.New("BlobSidecars must all be for the same block")
)

// LazilyPersistentStore is an implementation of AvailabilityStore to be used when batch syncing.
// This implementation will hold any blobs passed to Persist until the IsDataAvailable is called for their
// block, at which time they will undergo full verification and be saved to the disk.
type LazilyPersistentStore struct {
	store    *filesystem.BlobStorage
	cache    *cache
	verifier BlobBatchVerifier
}

var _ AvailabilityStore = &LazilyPersistentStore{}

// BlobBatchVerifier enables LazyAvailabilityStore to manage the verification process
// going from ROBlob->VerifiedROBlob, while avoiding the decision of which individual verifications
// to run and in what order. Since LazilyPersistentStore always tries to verify and save blobs only when
// they are all available, the interface takes a slice of blobs, enabling the implementation to optimize
// batch verification.
type BlobBatchVerifier interface {
	VerifiedROBlobs(ctx context.Context, sc []blocks.ROBlob) ([]blocks.VerifiedROBlob, error)
	MarkVerified(root [32]byte, slot primitives.Slot)
}

// NewLazilyPersistentStore creates a new LazilyPersistentStore. This constructor should always be used
// when creating a LazilyPersistentStore because it needs to initialize the cache under the hood.
func NewLazilyPersistentStore(store *filesystem.BlobStorage, verifier BlobBatchVerifier) *LazilyPersistentStore {
	return &LazilyPersistentStore{
		store:    store,
		cache:    newCache(),
		verifier: verifier,
	}
}

// Persist adds blobs to the working blob cache. Blobs stored in this cache will be persisted
// for at least as long as the node is running. Once IsDataAvailable succeeds, all blobs referenced
// by the given block are guaranteed to be persisted for the remainder of the retention period.
func (s *LazilyPersistentStore) Persist(current primitives.Slot, sc ...blocks.ROBlob) error {
	if len(sc) == 0 {
		return nil
	}
	if len(sc) > 1 {
		first := sc[0].BlockRoot()
		for i := 1; i < len(sc); i++ {
			if first != sc[i].BlockRoot() {
				return errMixedRoots
			}
		}
	}
	if !params.WithinDAPeriod(slots.ToEpoch(header(sc[0]).Slot), slots.ToEpoch(current)) {
		return nil
	}
	key := keyFromSidecar(sc[0])
	entry := s.cache.ensure(key)
	for i := range sc {
		if err := entry.stash(&sc[i]); err != nil {
			return err
		}
	}
	return nil
}

// IsDataAvailable returns nil if all the commitments in the given block are persisted to the db and have been verified.
// BlobSidecars already in the db are assumed to have been previously verified against the block.
func (s *LazilyPersistentStore) IsDataAvailable(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error {
	blockCommitments, err := commitmentsToCheck(b, current)
	if err != nil {
		return errors.Wrapf(err, "could check data availability for block %#x", b.Root())
	}
	if blockCommitments.count() == 0 {
		// If blockchain processing calls IsDataAvailable for a block it is valid as far as the verifier is concerned.
		// This func will early return for blocks that are pre-deneb or which do not have any commitments.
		// But first, we'll mark the block as verified for the rest of the batch
		// so that subsequent blocks can pass the parent-based validity checks.
		s.verifier.MarkVerified(b.Root(), b.Block().Slot())
		return nil
	}

	key := keyFromBlock(b)
	entry := s.cache.ensure(key)
	defer s.cache.delete(key)
	return s.daCheck(ctx, b.Root(), blockCommitments, entry)
}

func (s *LazilyPersistentStore) daCheck(ctx context.Context, root [32]byte, kc safeCommitmentArray, entry *cacheEntry) error {
	// Verify we have all the expected sidecars, and fail fast if any are missing or inconsistent.
	// We don't try to salvage problematic batches because this indicates a misbehaving peer and we'd rather
	// ignore their response and decrease their peer score.
	sidecars, err := entry.filter(root, kc)
	if err != nil {
		return errors.Wrap(err, "incomplete BlobSidecar batch")
	}
	// Do thorough verifications of each BlobSidecar for the block.
	// Same as above, we don't save BlobSidecars if there are any problems with the batch.
	vscs, err := s.verifier.VerifiedROBlobs(ctx, sidecars)
	if err != nil {
		return errors.Wrapf(err, "invalid BlobSidecars received for block %#x", root)
	}
	// Ensure that each BlobSidecar is written to disk.
	for i := range vscs {
		if err := s.store.Save(vscs[i]); err != nil {
			return errors.Wrapf(err, "failed to save BlobSidecar index %d for block %#x", vscs[i].Index, root)
		}
	}
	// All BlobSidecars are persisted - da check succeeds.
	return nil
}

func commitmentsToCheck(b blocks.ROBlock, current primitives.Slot) (safeCommitmentArray, error) {
	var ar safeCommitmentArray
	if b.Version() < version.Deneb {
		return ar, nil
	}
	// We are only required to check within MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS
	if !params.WithinDAPeriod(slots.ToEpoch(b.Block().Slot()), slots.ToEpoch(current)) {
		return ar, nil
	}
	kc, err := b.Block().Body().BlobKzgCommitments()
	if err != nil {
		return ar, err
	}
	if len(kc) > len(ar) {
		return ar, errIndexOutOfBounds
	}
	copy(ar[:], kc)
	return ar, nil
}

func header(s blocks.ROBlob) *ethpb.BeaconBlockHeader {
	return s.SignedBlockHeader.Header
}