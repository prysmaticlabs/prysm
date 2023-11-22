package verification

import (
	"context"

	"github.com/pkg/errors"
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
	RequireSlotBelowMaxDisparity
	RequireSlotBelowFinalized
	RequireValidProposerSignature
	RequireSidecarParentSeen
	RequireSidecarParentValid
	RequireSidecarParentSlotLower
	RequireSidecarDescendsFromFinalized
	RequireSidecarInclusionProven
	RequireSidecarBlobCommitmentProven
	RequireSidecarFirstSeen
	RequireSidecarProposerExpected
)

// GossipSidecarRequirements defines the set of requirements that BlobSidecars received on gossip
// must satisfy in order to upgrade an ROBlob to a VerifiedROBlob.
var GossipSidecarRequirements = []Requirement{
	RequireBlobIndexInBounds,
	RequireSlotBelowMaxDisparity,
	RequireSlotBelowFinalized,
	RequireValidProposerSignature,
	RequireSidecarParentSeen,
	RequireSidecarParentValid,
	RequireSidecarParentSlotLower,
	RequireSidecarDescendsFromFinalized,
	RequireSidecarInclusionProven,
	RequireSidecarBlobCommitmentProven,
	RequireSidecarFirstSeen,
	RequireSidecarProposerExpected,
}

var (
	ErrBlobInvalid            = errors.New("blob failed verification")
	ErrBlobIndexInBounds      = errors.Wrap(ErrBlobInvalid, "incorrect blob sidecar index")
	ErrSlotBelowMaxDisparity  = errors.Wrap(ErrBlobInvalid, "slot is too far in the future")
	ErrSlotBelowFinalized     = errors.Wrap(ErrBlobInvalid, "slot <= finalized checkpoint")
	ErrSidecarParentSeen      = errors.Wrap(ErrBlobInvalid, "parent root has not been seen")
	ErrSidecarParentValid     = errors.Wrap(ErrBlobInvalid, "parent block is not valid")
	ErrSidecarParentSlotLower = errors.Wrap(ErrBlobInvalid, "blob slot <= parent slot")
)

type BlobVerifier struct {
	*sharedResources
	ctx     context.Context
	results *results
	blob    blocks.ROBlob
}

// VerifiedROBlob "upgrades" the wrapped ROBlob to a VerifiedROBlob.
// If any of the verifications ran against the blob failed, or some required verifications
// were not run, an error will be returned.
func (bv *BlobVerifier) VerifiedROBlob() (blocks.VerifiedROBlob, error) {
	if bv.results.satisfied() {
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
		return ErrBlobIndexInBounds
	}
	return nil
}

func (bv *BlobVerifier) SlotBelowMaxDisparity() (err error) {
	defer bv.recordResult(RequireSlotBelowMaxDisparity, &err)
	if bv.clock.CurrentSlot() == bv.blob.Slot() {
		return nil
	}
	bt, err := slots.ToTime(uint64(bv.clock.GenesisTime().Second()), bv.blob.Slot())
	if err != nil {
		return errors.Wrapf(ErrSlotBelowMaxDisparity, "error computing slot time:%s", err.Error())
	}
	if bt.Sub(bv.clock.Now()) > params.BeaconNetworkConfig().MaximumGossipClockDisparity {
		return ErrSlotBelowMaxDisparity
	}
	return nil
}

func (bv *BlobVerifier) SlotBelowFinalized() (err error) {
	defer bv.recordResult(RequireSlotBelowFinalized, &err)
	fcp := bv.fc.FinalizedCheckpoint()
	fSlot, err := slots.EpochStart(fcp.Epoch)
	if err != nil {
		return errors.Wrapf(ErrSlotBelowFinalized, "error computing epoch start slot for finalized checkpoint (%d) %s", fcp.Epoch, err.Error())
	}
	if bv.blob.Slot() <= fSlot {
		return ErrSlotBelowFinalized
	}
	return nil
}

func (bv *BlobVerifier) ValidProposerSignature() (err error) {
	defer bv.recordResult(RequireValidProposerSignature, &err)
	blob := bv.blob
	sig := bytesutil.ToBytes96(blob.SignedBlockHeader.Signature)
	return bv.sc.VerifySignature(bv.ctx, sig, blob.BlockRoot(), blob.ParentRoot(), blob.ProposerIndex(), blob.Slot())
}

func (bv *BlobVerifier) SidecarParentSeen(badParent func([32]byte) bool) (err error) {
	defer bv.recordResult(RequireSidecarParentSeen, &err)
	if bv.fc.HasNode(bv.blob.ParentRoot()) {
		return nil
	}
	if badParent != nil && badParent(bv.blob.ParentRoot()) {
		return nil
	}
	return ErrSidecarParentSeen
}

func (bv *BlobVerifier) SidecareParentValid(badParent func([32]byte) bool) (err error) {
	defer bv.recordResult(RequireSidecarParentValid, &err)
	if badParent != nil && badParent(bv.blob.ParentRoot()) {
		return ErrSidecarParentValid
	}
	return nil
}

func (bv *BlobVerifier) SidecarParentSlotLower() (err error) {
	defer bv.recordResult(RequireSidecarParentSlotLower, &err)
	parentSlot, err := bv.fc.Slot(bv.blob.ParentRoot())
	if err != nil {
		return errors.Wrap(ErrSidecarParentSlotLower, "parent root not in forkchoice")
	}
	if parentSlot < bv.blob.Slot() {
		return nil
	}
	return ErrSidecarParentSlotLower
}

/*
func (bv *BlobVerifier) SidecarDescendsFromFinalized() (err error) {
	defer bv.recordResult(RequireSidecarDescendsFromFinalized, &err)
	return nil
}
*/

/*
	RequireSidecarDescendsFromFinalized
	RequireSidecarInclusionProven
	RequireSidecarBlobCommitmentProven
	RequireSidecarFirstSeen
	RequireSidecarProposerExpected
*/
