package verify

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/verification"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

var (
	errBlobVerification   = errors.New("unable to verify blobs")
	errColumnVerification = errors.New("unable to verify column")

	ErrIncorrectBlobIndex   = errors.New("incorrect blob index")
	ErrIncorrectColumnIndex = errors.New("incorrect column index")

	ErrBlobBlockMisaligned   = errors.Wrap(errBlobVerification, "root of block header in blob sidecar does not match block root")
	ErrColumnBlockMisaligned = errors.Wrap(errColumnVerification, "root of block header in column sidecar does not match block root")

	ErrMismatchedBlobCommitments   = errors.Wrap(errBlobVerification, "commitments at given slot, root and index do not match")
	ErrMismatchedColumnCommitments = errors.Wrap(errColumnVerification, "commitments at given slot, root and index do not match")
)

// BlobAlignsWithBlock verifies if the blob aligns with the block.
func BlobAlignsWithBlock(blob blocks.ROBlob, block blocks.ROBlock) error {
	if block.Version() < version.Deneb {
		return nil
	}
	if blob.Index >= fieldparams.MaxBlobsPerBlock {
		return errors.Wrapf(ErrIncorrectBlobIndex, "index %d exceeds MAX_BLOBS_PER_BLOCK %d", blob.Index, fieldparams.MaxBlobsPerBlock)
	}

	if blob.BlockRoot() != block.Root() {
		return ErrBlobBlockMisaligned
	}

	// Verify commitment byte values match
	// TODO: verify commitment inclusion proof - actually replace this with a better rpc blob verification stack altogether.
	commits, err := block.Block().Body().BlobKzgCommitments()
	if err != nil {
		return err
	}
	blockCommitment := bytesutil.ToBytes48(commits[blob.Index])
	blobCommitment := bytesutil.ToBytes48(blob.KzgCommitment)
	if blobCommitment != blockCommitment {
		return errors.Wrapf(ErrMismatchedBlobCommitments, "commitment %#x != block commitment %#x, at index %d for block root %#x at slot %d ", blobCommitment, blockCommitment, blob.Index, block.Root(), blob.Slot())
	}
	return nil
}

func ColumnAlignsWithBlock(col blocks.RODataColumn, block blocks.ROBlock, colVerifier verification.NewColumnVerifier) error {
	// Exit early if the block is not at least a Deneb block.
	if block.Version() < version.Deneb {
		return nil
	}

	// Check if the block root in the column sidecar matches the block root.
	if col.BlockRoot() != block.Root() {
		return ErrColumnBlockMisaligned
	}

	// Verify commitment byte values match
	commitments, err := block.Block().Body().BlobKzgCommitments()
	if err != nil {
		return errors.Wrap(err, "blob KZG commitments")
	}

	if !reflect.DeepEqual(commitments, col.KzgCommitments) {
		return errors.Wrapf(
			ErrMismatchedColumnCommitments,
			"commitment %#v != block commitment %#v for block root %#x at slot %d ",
			col.KzgCommitments,
			commitments,
			block.Root(),
			col.Slot(),
		)
	}

	vf := colVerifier(col, verification.InitsyncColumnSidecarRequirements)
	if err := vf.DataColumnIndexInBounds(); err != nil {
		return errors.Wrap(err, "data column index out of bounds")
	}

	// Filter out columns which did not pass the KZG inclusion proof verification.
	if err := vf.SidecarInclusionProven(); err != nil {
		return errors.Wrap(err, "inclusion proof verification")
	}

	// Filter out columns which did not pass the KZG proof verification.
	if err := vf.SidecarKzgProofVerified(); err != nil {
		return errors.Wrap(err, "KZG proof verification")
	}

	return nil
}
