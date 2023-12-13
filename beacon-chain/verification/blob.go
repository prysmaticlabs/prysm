package verification

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/runtime/logging"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
)

const (
	RequireBlobIndexInBounds Requirement = iota
	RequireSlotNotTooEarly
	RequireSlotAboveFinalized
	RequireValidProposerSignature
	RequireSidecarParentSeen
	RequireSidecarParentValid
	RequireSidecarParentSlotLower
	RequireSidecarDescendsFromFinalized
	RequireSidecarInclusionProven
	RequireSidecarKzgProofVerified
	RequireSidecarProposerExpected
)

// GossipSidecarRequirements defines the set of requirements that BlobSidecars received on gossip
// must satisfy in order to upgrade an ROBlob to a VerifiedROBlob.
var GossipSidecarRequirements = []Requirement{
	RequireBlobIndexInBounds,
	RequireSlotNotTooEarly,
	RequireSlotAboveFinalized,
	RequireValidProposerSignature,
	RequireSidecarParentSeen,
	RequireSidecarParentValid,
	RequireSidecarParentSlotLower,
	RequireSidecarDescendsFromFinalized,
	RequireSidecarInclusionProven,
	RequireSidecarKzgProofVerified,
	RequireSidecarProposerExpected,
}

var (
	ErrBlobInvalid = errors.New("blob failed verification")
	// ErrBlobIndexInvalid means RequireBlobIndexInBounds failed.
	ErrBlobIndexInvalid = errors.Wrap(ErrBlobInvalid, "incorrect blob sidecar index")
	// ErrSlotTooEarly means RequireSlotNotTooEarly failed.
	ErrSlotTooEarly = errors.Wrap(ErrBlobInvalid, "slot is too far in the future")
	// ErrSlotNotAfterFinalized means RequireSlotAboveFinalized failed.
	ErrSlotNotAfterFinalized = errors.Wrap(ErrBlobInvalid, "slot <= finalized checkpoint")
	// ErrInvalidProposerSignature means RequireValidProposerSignature failed.
	ErrInvalidProposerSignature = errors.Wrap(ErrBlobInvalid, "proposer signature could not be verified")
	// ErrSidecarParentNotSeen means RequireSidecarParentSeen failed.
	ErrSidecarParentNotSeen = errors.Wrap(ErrBlobInvalid, "parent root has not been seen")
	// ErrSidecarParentInvalid means RequireSidecarParentValid failed.
	ErrSidecarParentInvalid = errors.Wrap(ErrBlobInvalid, "parent block is not valid")
	// ErrSlotNotAfterParent means RequireSidecarParentSlotLower failed.
	ErrSlotNotAfterParent = errors.Wrap(ErrBlobInvalid, "slot <= slot")
	// ErrSidecarNotFinalizedDescendent means RequireSidecarDescendsFromFinalized failed.
	ErrSidecarNotFinalizedDescendent = errors.Wrap(ErrBlobInvalid, "blob parent is not descended from the finalized block")
	// ErrSidecarInclusionProofInvalid means RequireSidecarInclusionProven failed.
	ErrSidecarInclusionProofInvalid = errors.Wrap(ErrBlobInvalid, "sidecar inclusion proof verification failed")
	// ErrSidecarKzgProofInvalid means RequireSidecarKzgProofVerified failed.
	ErrSidecarKzgProofInvalid = errors.Wrap(ErrBlobInvalid, "sidecar kzg commitment proof verification failed")
	// ErrSidecarUnexpectedProposer means RequireSidecarProposerExpected failed.
	ErrSidecarUnexpectedProposer = errors.Wrap(ErrBlobInvalid, "sidecar was not proposed by the expected proposer_index")
)

type BlobVerifier struct {
	*sharedResources
	results              *results
	blob                 blocks.ROBlob
	parent               state.BeaconState
	verifyBlobCommitment roblobCommitmentVerifier
}

type roblobCommitmentVerifier func(blocks.ROBlob) error

// VerifiedROBlob "upgrades" the wrapped ROBlob to a VerifiedROBlob.
// If any of the verifications ran against the blob failed, or some required verifications
// were not run, an error will be returned.
func (bv *BlobVerifier) VerifiedROBlob() (blocks.VerifiedROBlob, error) {
	if bv.results.allSatisfied() {
		return blocks.NewVerifiedROBlob(bv.blob), nil
	}
	return blocks.VerifiedROBlob{}, bv.results.errors(ErrBlobInvalid)
}

func (bv *BlobVerifier) recordResult(req Requirement, err *error) {
	if err == nil || *err == nil {
		bv.results.record(req, nil)
		return
	}
	bv.results.record(req, *err)
}

// BlobIndexInBounds represents the follow spec verification:
// [REJECT] The sidecar's index is consistent with MAX_BLOBS_PER_BLOCK -- i.e. blob_sidecar.index < MAX_BLOBS_PER_BLOCK.
func (bv *BlobVerifier) BlobIndexInBounds() (err error) {
	defer bv.recordResult(RequireBlobIndexInBounds, &err)
	if bv.blob.Index >= fieldparams.MaxBlobsPerBlock {
		log.WithFields(logging.BlobFields(bv.blob)).Debug("Sidecar index > MAX_BLOBS_PER_BLOCK")
		return ErrBlobIndexInvalid
	}
	return nil
}

// SlotNotTooEarly represents the spec verification:
// [IGNORE] The sidecar is not from a future slot (with a MAXIMUM_GOSSIP_CLOCK_DISPARITY allowance)
// -- i.e. validate that block_header.slot <= current_slot
func (bv *BlobVerifier) SlotNotTooEarly() (err error) {
	defer bv.recordResult(RequireSlotNotTooEarly, &err)
	if bv.clock.CurrentSlot() == bv.blob.Slot() {
		return nil
	}
	// subtract the max clock disparity from the start slot time
	validAfter := bv.clock.SlotStart(bv.blob.Slot()).Add(-1 * params.BeaconNetworkConfig().MaximumGossipClockDisparity)
	// If the difference between now and gt is greater than maximum clock disparity, the block is too far in the future.
	if bv.clock.Now().Before(validAfter) {
		return ErrSlotTooEarly
	}
	return nil
}

// SlotAboveFinalized represents the spec verification:
// [IGNORE] The sidecar is from a slot greater than the latest finalized slot
// -- i.e. validate that block_header.slot > compute_start_slot_at_epoch(state.finalized_checkpoint.epoch)
func (bv *BlobVerifier) SlotAboveFinalized() (err error) {
	defer bv.recordResult(RequireSlotAboveFinalized, &err)
	fcp := bv.fc.FinalizedCheckpoint()
	fSlot, err := slots.EpochStart(fcp.Epoch)
	if err != nil {
		return errors.Wrapf(ErrSlotNotAfterFinalized, "error computing epoch start slot for finalized checkpoint (%d) %s", fcp.Epoch, err.Error())
	}
	if bv.blob.Slot() <= fSlot {
		return ErrSlotNotAfterFinalized
	}
	return nil
}

// ValidProposerSignature represents the spec verification:
// [REJECT] The proposer signature of blob_sidecar.signed_block_header,
// is valid with respect to the block_header.proposer_index pubkey.
func (bv *BlobVerifier) ValidProposerSignature(ctx context.Context) (err error) {
	defer bv.recordResult(RequireValidProposerSignature, &err)
	sd := blobToSignatureData(bv.blob)
	// First check if there is a cached verification that can be reused.
	seen, err := bv.sc.SignatureVerified(sd)
	if seen {
		if err != nil {
			log.WithFields(logging.BlobFields(bv.blob)).WithError(err).Debug("reusing failed proposer signature validation from cache")
			return ErrInvalidProposerSignature
		}
		return nil
	}

	// retrieve the parent state to fallback to full verification
	parent, err := bv.parentState(ctx)
	if err != nil {
		log.WithFields(logging.BlobFields(bv.blob)).WithError(err).Debug("could not replay parent state for blob signature verification")
		return ErrInvalidProposerSignature
	}
	// Full verification, which will subsequently be cached for anything sharing the signature cache.
	if err := bv.sc.VerifySignature(sd, parent); err != nil {
		log.WithFields(logging.BlobFields(bv.blob)).WithError(err).Debug("signature verification failed")
		return ErrInvalidProposerSignature
	}
	return nil
}

// SidecarParentSeen represents the spec verification:
// [IGNORE] The sidecar's block's parent (defined by block_header.parent_root) has been seen
// (via both gossip and non-gossip sources) (a client MAY queue sidecars for processing once the parent block is retrieved).
func (bv *BlobVerifier) SidecarParentSeen(badParent func([32]byte) bool) (err error) {
	defer bv.recordResult(RequireSidecarParentSeen, &err)
	if bv.fc.HasNode(bv.blob.ParentRoot()) {
		return nil
	}
	if badParent != nil && badParent(bv.blob.ParentRoot()) {
		return nil
	}
	return ErrSidecarParentNotSeen
}

// SidecarParentValid represents the spec verification:
// [REJECT] The sidecar's block's parent (defined by block_header.parent_root) passes validation.
func (bv *BlobVerifier) SidecarParentValid(badParent func([32]byte) bool) (err error) {
	defer bv.recordResult(RequireSidecarParentValid, &err)
	if badParent != nil && badParent(bv.blob.ParentRoot()) {
		return ErrSidecarParentInvalid
	}
	return nil
}

// SidecarParentSlotLower represents the spec verification:
// [REJECT] The sidecar is from a higher slot than the sidecar's block's parent (defined by block_header.parent_root).
func (bv *BlobVerifier) SidecarParentSlotLower() (err error) {
	defer bv.recordResult(RequireSidecarParentSlotLower, &err)
	parentSlot, err := bv.fc.Slot(bv.blob.ParentRoot())
	if err != nil {
		return errors.Wrap(ErrSlotNotAfterParent, "parent root not in forkchoice")
	}
	if parentSlot >= bv.blob.Slot() {
		return ErrSlotNotAfterParent
	}
	return nil
}

// SidecarDescendsFromFinalized represents the spec verification:
// [REJECT] The current finalized_checkpoint is an ancestor of the sidecar's block
// -- i.e. get_checkpoint_block(store, block_header.parent_root, store.finalized_checkpoint.epoch) == store.finalized_checkpoint.root.
func (bv *BlobVerifier) SidecarDescendsFromFinalized() (err error) {
	defer bv.recordResult(RequireSidecarDescendsFromFinalized, &err)
	if !bv.fc.IsCanonical(bv.blob.ParentRoot()) {
		return ErrSidecarNotFinalizedDescendent
	}
	return nil
}

// SidecarInclusionProven represents the spec verification:
// [REJECT] The sidecar's inclusion proof is valid as verified by verify_blob_sidecar_inclusion_proof(blob_sidecar).
func (bv *BlobVerifier) SidecarInclusionProven() (err error) {
	defer bv.recordResult(RequireSidecarInclusionProven, &err)
	if err := blocks.VerifyKZGInclusionProof(bv.blob); err != nil {
		log.WithError(err).WithFields(logging.BlobFields(bv.blob)).Debug("sidecar inclusion proof verification failed")
		return ErrSidecarInclusionProofInvalid
	}
	return nil
}

// SidecarKzgProofVerified represents the spec verification:
// [REJECT] The sidecar's blob is valid as verified by
// verify_blob_kzg_proof(blob_sidecar.blob, blob_sidecar.kzg_commitment, blob_sidecar.kzg_proof).
func (bv *BlobVerifier) SidecarKzgProofVerified() (err error) {
	defer bv.recordResult(RequireSidecarKzgProofVerified, &err)
	if err := bv.verifyBlobCommitment(bv.blob); err != nil {
		log.WithError(err).WithFields(logging.BlobFields(bv.blob)).Debug("kzg commitment proof verification failed")
		return ErrSidecarKzgProofInvalid
	}
	return nil
}

// SidecarProposerExpected represents the spec verification:
// [REJECT] The sidecar is proposed by the expected proposer_index for the block's slot
// in the context of the current shuffling (defined by block_header.parent_root/block_header.slot).
// If the proposer_index cannot immediately be verified against the expected shuffling, the sidecar MAY be queued
// for later processing while proposers for the block's branch are calculated -- in such a case do not REJECT, instead IGNORE this message.
func (bv *BlobVerifier) SidecarProposerExpected(ctx context.Context) (err error) {
	defer bv.recordResult(RequireSidecarProposerExpected, &err)
	idx, cached := bv.pc.Proposer(bv.blob.ParentRoot(), bv.blob.Slot())
	if !cached {
		pst, err := bv.parentState(ctx)
		if err != nil {
			log.WithError(err).WithFields(logging.BlobFields(bv.blob)).Debug("state replay to parent_root failed")
			return ErrSidecarUnexpectedProposer
		}
		idx, err = bv.pc.ComputeProposer(ctx, bv.blob.ParentRoot(), bv.blob.Slot(), pst)
		if err != nil {
			log.WithError(err).WithFields(logging.BlobFields(bv.blob)).Debug("error computing proposer index from parent state")
			return ErrSidecarUnexpectedProposer
		}
	}
	if idx != bv.blob.ProposerIndex() {
		log.WithError(ErrSidecarUnexpectedProposer).
			WithFields(logging.BlobFields(bv.blob)).WithField("expected_proposer", idx).
			Debug("unexpected blob proposer")
		return ErrSidecarUnexpectedProposer
	}
	return nil
}

func (bv *BlobVerifier) parentState(ctx context.Context) (state.BeaconState, error) {
	if bv.parent != nil {
		return bv.parent, nil
	}
	st, err := bv.sr.StateByRoot(ctx, bv.blob.ParentRoot())
	if err != nil {
		return nil, err
	}
	bv.parent = st
	return bv.parent, nil
}

func blobToSignatureData(b blocks.ROBlob) SignatureData {
	return SignatureData{
		Root:      b.BlockRoot(),
		Parent:    b.ParentRoot(),
		Signature: bytesutil.ToBytes96(b.SignedBlockHeader.Signature),
		Proposer:  b.ProposerIndex(),
		Slot:      b.Slot(),
	}
}
