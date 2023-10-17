package das

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	errors "github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/kzg"
	"github.com/prysmaticlabs/prysm/v4/cache/nonblocking"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
)

var (
	errCachedCommitmentMismatch = errors.New("previously verified commitments do not match those in block")
	// concurrentBlockFetchers * blocks per request is used to size the LRU so that we can have one cache
	// for many worker goroutines without them evicting each others results.
	// TODO: figure out how to determine the max number of init sync workers.
	concurrentBlockFetchers = 10
)

// AvailabilityStore describes a component that can verify and save sidecars for a given block, and confirm previously
// verified and saved sidecars.
type AvailabilityStore interface {
	VerifyAvailability(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error
	SaveIfAvailable(ctx context.Context, current primitives.Slot, b blocks.BlockWithVerifiedBlobs) error
}

type CachingDBVerifiedStore struct {
	sync.RWMutex
	db    BlobsDB
	cache *nonblocking.LRU[[32]byte, [][]byte]
}

var _ AvailabilityStore = &CachingDBVerifiedStore{}

func NewCachingDBVerifiedStore(db BlobsDB) *CachingDBVerifiedStore {
	discardev := func([32]byte, [][]byte) {}
	size := int(params.BeaconNetworkConfig().MaxRequestBlocks) * concurrentBlockFetchers
	cache, err := nonblocking.NewLRU[[32]byte, [][]byte](size, discardev)
	if err != nil {
		panic(err)
	}
	return &CachingDBVerifiedStore{
		db:    db,
		cache: cache,
	}
}

func (s *CachingDBVerifiedStore) VerifyAvailability(ctx context.Context, current primitives.Slot, b blocks.ROBlock) error {
	c := commitmentsToCheck(b, current)
	if len(c) == 0 {
		return nil
	}
	// get the previously verified commitments
	vc, ok := s.cache.Get(b.Root())
	if !ok {
		return s.databaseDACheck(ctx, current, b.Root(), c)
	}
	// check commitments to make sure that they match
	if len(c) != len(vc) {
		return errors.Wrapf(errCachedCommitmentMismatch, "cache=%d, block=%d", len(vc), len(c))
	}
	for i := range vc {
		// check each commitment in the block against value previously validated and saved.
		if !bytes.Equal(vc[i], c[i]) {
			return errors.Wrapf(errCachedCommitmentMismatch, "commitment %#x at index %d does not match cache %#x", c[i], i, vc[i])
		}
	}
	// all commitments match
	return nil
}

func (s *CachingDBVerifiedStore) SaveIfAvailable(ctx context.Context, current primitives.Slot, bwb blocks.BlockWithVerifiedBlobs) error {
	b := bwb.Block
	c := commitmentsToCheck(b, current)
	if len(c) == 0 {
		return nil
	}
	if err := kzg.IsDataAvailable(c, bwb.Blobs); err != nil {
		return errors.Wrapf(err, "kzg.IsDataAvailable check failed for %#x", b.Root())
	}
	if err := s.db.SaveBlobSidecar(ctx, bwb.Blobs); err != nil {
		return errors.Wrapf(err, "error while trying to save verified blob sidecars for root %#x", b.Root())
	}
	// cache the commitments that matched so that we can confirm them cheaply while they remain in cache.
	s.cache.Add(b.Root(), c)
	return nil
}

func (s *CachingDBVerifiedStore) databaseDACheck(ctx context.Context, current primitives.Slot, root [32]byte, cmts [][]byte) error {
	log.WithField("root", fmt.Sprintf("%#x", root)).Warn("Falling back to database DA check")
	sidecars, err := s.db.BlobSidecarsByRoot(ctx, root)
	if err != nil {
		return errors.Wrap(err, "could not get blob sidecars")
	}
	if err := kzg.IsDataAvailable(cmts, sidecars); err != nil {
		return err
	}
	s.cache.Add(root, cmts)
	return nil
}

type BlobsDB interface {
	BlobSidecarsByRoot(ctx context.Context, beaconBlockRoot [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error)
	SaveBlobSidecar(ctx context.Context, sidecars []*ethpb.BlobSidecar) error
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
