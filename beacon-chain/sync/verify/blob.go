package verify

import (
	"github.com/pkg/errors"
	field_params "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
)

var (
	errBlobVerification          = errors.New("unable to verify blobs")
	ErrMismatchedBlobBlockRoot   = errors.Wrap(errBlobVerification, "BlockRoot in BlobSidecar does not match the expected root")
	ErrMismatchedBlobBlockSlot   = errors.Wrap(errBlobVerification, "BlockSlot in BlobSidecar does not match the expected slot")
	ErrMismatchedBlobCommitments = errors.Wrap(errBlobVerification, "commitments at given slot, root and index do not match")
	ErrMismatchedProposerIndex   = errors.Wrap(errBlobVerification, "proposer index does not match")
	ErrIncorrectBlobIndex        = errors.Wrap(errBlobVerification, "incorrect blob index")
)

// BlobAlignsWithBlock verifies if the blob aligns with the block.
func BlobAlignsWithBlock(blobSidecar blocks.ROBlob, block blocks.ROBlock) error {
	if block.Version() < version.Deneb {
		return nil
	}

	commits, err := block.Block().Body().BlobKzgCommitments()
	if err != nil {
		return err
	}

	if len(commits) == 0 {
		return nil
	}

	if blobSidecar.Index >= field_params.MaxBlobsPerBlock {
		return errors.Wrapf(ErrIncorrectBlobIndex, "blobSidecar index %d >= max blobs per block %d", blobSidecar.Index, field_params.MaxBlobsPerBlock)
	}

	if blobSidecar.BlockRoot() != block.Root() {
		return errors.Wrapf(ErrMismatchedBlobBlockRoot, "blobSidecar root %#x != block root %#x", blobSidecar.BlockRoot(), block.Root())
	}

	// TODO: Verify blobSidecar to kzg commitment is correct
	// TODO: Verify sidecar's inclusion proof is correct

	return nil
}
