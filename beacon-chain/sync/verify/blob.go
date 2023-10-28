package verify

import (
	"github.com/pkg/errors"
	field_params "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
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
func BlobAlignsWithBlock(blob *ethpb.BlobSidecar, block blocks.ROBlock) error {
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

	if blob.Index >= field_params.MaxBlobsPerBlock {
		return errors.Wrapf(ErrIncorrectBlobIndex, "blob index %d >= max blobs per block %d", blob.Index, field_params.MaxBlobsPerBlock)
	}

	header := blob.SignedBlockHeader
	headerRoot, err := header.Header.HashTreeRoot()
	if err != nil {
		return err
	}
	// Verify block and parent roots
	if headerRoot != block.Root() {
		return errors.Wrapf(ErrMismatchedBlobBlockRoot, "block root %#x != BlobSidecar.BlockRoot %#x at slot %d", block.Root(), headerRoot, header.Header.Slot)
	}

	// Verify commitment
	blockCommitment := bytesutil.ToBytes48(commits[blob.Index])
	blobCommitment := bytesutil.ToBytes48(blob.KzgCommitment)
	if blobCommitment != blockCommitment {
		return errors.Wrapf(ErrMismatchedBlobCommitments, "commitment %#x != block commitment %#x, at index %d for block root %#x at slot %d ", blobCommitment, blockCommitment, blob.Index, headerRoot, header.Header.Slot)
	}

	return nil
}
