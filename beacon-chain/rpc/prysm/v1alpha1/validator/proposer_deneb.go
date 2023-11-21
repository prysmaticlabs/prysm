package validator

import (
	"sync"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
)

var bundleCache = &blobsBundleCache{
	blobs: make(map[primitives.Slot]*enginev1.BlobsBundle),
}

// BlobsBundleCache holds the KZG commitments and other relevant sidecar data for a local beacon block.
type blobsBundleCache struct {
	blobs map[primitives.Slot]*enginev1.BlobsBundle
	sync.Mutex
}

// add adds a blobs bundle to the cache.
// same slot overwrites the previous bundle.
func (c *blobsBundleCache) add(slot primitives.Slot, bundle *enginev1.BlobsBundle) {
	c.Lock()
	c.blobs[slot] = bundle
	c.Unlock()

	// Trigger pruning in the background
	go c.prune(slot)
}

// get gets a blobs bundle from the cache.
func (c *blobsBundleCache) get(slot primitives.Slot) *enginev1.BlobsBundle {
	c.Lock()
	blobs := c.blobs[slot]
	c.Unlock()

	// Trigger pruning in the background
	go c.prune(slot)

	return blobs
}

// prune removes blobs bundles from the cache that are equal or older than the given slot.
func (c *blobsBundleCache) prune(minSlot primitives.Slot) {
	c.Lock()
	defer c.Unlock()
	for s := range c.blobs {
		if s < minSlot {
			delete(c.blobs, s)
		}
	}
}

// coverts a blobs bundle to a sidecar format.
func blobsBundleToSidecars(bundle *enginev1.BlobsBundle, blk interfaces.ReadOnlyBeaconBlock) ([]*ethpb.DeprecatedBlobSidecar, error) {
	if blk.Version() < version.Deneb {
		return nil, nil
	}
	if bundle == nil || len(bundle.KzgCommitments) == 0 {
		return nil, nil
	}
	r, err := blk.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	pr := blk.ParentRoot()

	sidecars := make([]*ethpb.DeprecatedBlobSidecar, len(bundle.Blobs))
	for i := 0; i < len(bundle.Blobs); i++ {
		sidecars[i] = &ethpb.DeprecatedBlobSidecar{
			BlockRoot:       r[:],
			Index:           uint64(i),
			Slot:            blk.Slot(),
			BlockParentRoot: pr[:],
			ProposerIndex:   blk.ProposerIndex(),
			Blob:            bundle.Blobs[i],
			KzgCommitment:   bundle.KzgCommitments[i],
			KzgProof:        bundle.Proofs[i],
		}
	}

	return sidecars, nil
}

// coverts a blinds blobs bundle to a sidecar format.
func blindBlobsBundleToSidecars(bundle *enginev1.BlindedBlobsBundle, blk interfaces.ReadOnlyBeaconBlock) ([]*ethpb.BlindedBlobSidecar, error) {
	if blk.Version() < version.Deneb {
		return nil, nil
	}
	if bundle == nil || len(bundle.KzgCommitments) == 0 {
		return nil, nil
	}
	r, err := blk.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	pr := blk.ParentRoot()

	sidecars := make([]*ethpb.BlindedBlobSidecar, len(bundle.BlobRoots))
	for i := 0; i < len(bundle.BlobRoots); i++ {
		sidecars[i] = &ethpb.BlindedBlobSidecar{
			BlockRoot:       r[:],
			Index:           uint64(i),
			Slot:            blk.Slot(),
			BlockParentRoot: pr[:],
			ProposerIndex:   blk.ProposerIndex(),
			BlobRoot:        bundle.BlobRoots[i],
			KzgCommitment:   bundle.KzgCommitments[i],
			KzgProof:        bundle.Proofs[i],
		}
	}

	return sidecars, nil
}
