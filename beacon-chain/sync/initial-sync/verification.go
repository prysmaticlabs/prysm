package initialsync

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/kzg"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/das"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/verification"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

func newBlobVerifierFromInitializer(ini *verification.Initializer) verification.NewBlobVerifier {
	return func(b blocks.ROBlob, reqs []verification.Requirement) verification.BlobVerifier {
		return ini.NewBlobVerifier(b, reqs)
	}
}

func newBlobBatchVerifier(newVerifier verification.NewBlobVerifier) *BlobBatchVerifier {
	return &BlobBatchVerifier{
		verified:    make(map[[32]byte]primitives.Slot),
		verifyKzg:   kzg.Verify,
		newVerifier: newVerifier,
	}
}

type kzgVerifier func(b ...blocks.ROBlob) error

// BlobBatchVerifier solves problems that come from verifying batches of blobs from RPC.
// First: we only update forkchoice after the entire batch has completed, so the n+1 elements in the batch
// won't be in forkchoice yet.
// Second: it is more efficient to batch some verifications, like kzg commitment verification. Batch adds an additional
// method to BlobVerifier to verify the kzg commitments of all blob sidecars for a block together, then using the cached
// result of the batch verification when verifying the individual blobs.
type BlobBatchVerifier struct {
	verifyKzg   kzgVerifier
	newVerifier verification.NewBlobVerifier
	verified    map[[32]byte]primitives.Slot
}

// MarkVerified is exported so that blobs without commitments can be marked valid.
// This allows a user of the BlobBatchVerifier to early return while still keeping
// track of previous blocks in the batch.
func (batch *BlobBatchVerifier) MarkVerified(root [32]byte, slot primitives.Slot) {
	batch.verified[root] = slot
}

var _ das.BlobBatchVerifier = &BlobBatchVerifier{}

func (batch *BlobBatchVerifier) VerifiedROBlobs(ctx context.Context, scs []blocks.ROBlob) ([]blocks.VerifiedROBlob, error) {
	if len(scs) == 0 {
		return nil, nil
	}
	if err := batch.verifyKzg(scs...); err != nil {
		return nil, err
	}
	vs := make([]blocks.VerifiedROBlob, len(scs))
	for i := range scs {
		vb, err := batch.verifyOneBlob(ctx, scs[i])
		if err != nil {
			return nil, err
		}
		vs[i] = vb
	}
	return vs, nil
}

func (batch *BlobBatchVerifier) verifyOneBlob(ctx context.Context, sc blocks.ROBlob) (blocks.VerifiedROBlob, error) {
	vb := blocks.VerifiedROBlob{}
	bv := batch.newVerifier(sc, verification.InitsyncSidecarRequirements)
	// We can satisfy this immediately because VerifiedROBlobs always verifies commitments for all blobs in the batch
	// before calling verifyOneBlob.
	bv.SatisfyRequirement(verification.RequireSidecarKzgProofVerified)

	if err := bv.BlobIndexInBounds(); err != nil {
		return vb, err
	}
	/*
		if err := bv.NotFromFutureSlot(); err != nil {
			return vb, err
		}
		if err := bv.SlotAboveFinalized(); err != nil {
			return vb, err
		}
	*/

	/*
		// If we've previously verified a sidecar for a given block root, we don't need to perform these other checks,
		// because the matching block root ensures the slot and parent root match,
		// making the checks in the 'else' branch redundant.
		_, verified := batch.verified[sc.BlockRoot()]
		// Since we are processing in batches, it is not possible to use methods that look at forkchoice data, which is
		// only updated at the end of the batch. But, if this method has previously seen a sidecar for the parent
		// and completely verified it, we know all these properties hold true for the child as well, as long as the parent's
		// slot satisfies the following inequality. This code assumes responsibility for ensuring this
		// assumption is correct using SatisfyRequirement to skip the verifier methods.
		parentSlot, parentVerified := batch.verified[sc.ParentRoot()]
		if verified || parentVerified && parentSlot < sc.Slot() {
			bv.SatisfyRequirement(verification.RequireSidecarParentSeen)
			bv.SatisfyRequirement(verification.RequireSidecarParentValid)
			bv.SatisfyRequirement(verification.RequireSidecarParentSlotLower)
			bv.SatisfyRequirement(verification.RequireSidecarDescendsFromFinalized)
		} else {
			if err := bv.SidecarParentSeen(nil); err != nil {
				return vb, err
			}
			if err := bv.SidecarParentValid(nil); err != nil {
				return vb, err
			}
			if err := bv.SidecarParentSlotLower(); err != nil {
				return vb, err
			}
			if err := bv.SidecarDescendsFromFinalized(); err != nil {
				return vb, err
			}
		}
	*/

	if err := bv.SidecarInclusionProven(); err != nil {
		return vb, err
	}

	vb, err := bv.VerifiedROBlob()
	if err == nil {
		batch.MarkVerified(sc.BlockRoot(), sc.Slot())
	}
	return vb, err
}
