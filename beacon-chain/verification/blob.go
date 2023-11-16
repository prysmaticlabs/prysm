package verification

import (
	"fmt"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/sirupsen/logrus"
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
	ErrBlobInvalid       = errors.New("blob failed verification")
	ErrBlobIndexInBounds = errors.Wrap(ErrBlobInvalid, "incorrect blob sidecar index")
)

type BlobVerifier struct {
	shared  *sharedResources
	results *results
	blob    blocks.ROBlob
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
		bv.logWithFields().Debug("Sidecar index > MAX_BLOBS_PER_BLOCK")
		return ErrBlobIndexInBounds
	}
	return nil
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

func (bv *BlobVerifier) logWithFields() *logrus.Entry {
	return log.WithFields(logrus.Fields{
		"slot":          bv.blob.Slot(),
		"proposerIndex": bv.blob.ProposerIndex(),
		"blockRoot":     fmt.Sprintf("%#x", bv.blob.BlockRoot()),
		"kzgCommitment": fmt.Sprintf("%#x", bv.blob.KzgCommitment),
		"index":         bv.blob.Index,
	})
}
