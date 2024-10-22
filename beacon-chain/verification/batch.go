package verification

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/kzg"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/peerdas"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
)

var (
	// ErrBatchSignatureMismatch is returned by VerifiedROBlobs when any of the blobs in the batch have a signature
	// which does not match the signature for the block with a corresponding root.
	ErrBatchSignatureMismatch = errors.New("Sidecar block header signature does not match signed block")
	// ErrBatchBlockRootMismatch is returned by VerifiedROBlobs in the scenario where the root of the given signed block
	// does not match the block header in one of the corresponding sidecars.
	ErrBatchBlockRootMismatch = errors.New("Sidecar block header root does not match signed block")
)

// NewBlobBatchVerifier initializes a blob batch verifier. It requires the caller to correctly specify
// verification Requirements and to also pass in a NewBlobVerifier, which is a callback function that
// returns a new BlobVerifier for handling a single blob in the batch.
func NewBlobBatchVerifier(newVerifier NewBlobVerifier, reqs []Requirement) *BlobBatchVerifier {
	return &BlobBatchVerifier{
		verifyKzg:   kzg.Verify,
		newVerifier: newVerifier,
		reqs:        reqs,
	}
}

// BlobBatchVerifier solves problems that come from verifying batches of blobs from RPC.
// First: we only update forkchoice after the entire batch has completed, so the n+1 elements in the batch
// won't be in forkchoice yet.
// Second: it is more efficient to batch some verifications, like kzg commitment verification. Batch adds a
// method to BlobVerifier to verify the kzg commitments of all blob sidecars for a block together, then using the cached
// result of the batch verification when verifying the individual blobs.
type BlobBatchVerifier struct {
	verifyKzg   roblobCommitmentVerifier
	newVerifier NewBlobVerifier
	reqs        []Requirement
}

// VerifiedROBlobs satisfies the das.BlobBatchVerifier interface, used by das.AvailabilityStore.
func (batch *BlobBatchVerifier) VerifiedROBlobs(_ context.Context, blk blocks.ROBlock, scs []blocks.ROBlob) ([]blocks.VerifiedROBlob, error) {
	if len(scs) == 0 {
		return nil, nil
	}
	blkSig := blk.Signature()
	// We assume the proposer is validated wrt the block in batch block processing before performing the DA check.
	// So at this stage we just need to make sure the value being signed and signature bytes match the block.
	for i := range scs {
		blobSig := bytesutil.ToBytes96(scs[i].SignedBlockHeader.Signature)
		if blkSig != blobSig {
			return nil, ErrBatchSignatureMismatch
		}
		// Extra defensive check to make sure the roots match. This should be unnecessary in practice since the root from
		// the block should be used as the lookup key into the cache of sidecars.
		if blk.Root() != scs[i].BlockRoot() {
			return nil, ErrBatchBlockRootMismatch
		}
	}
	// Verify commitments for all blobs at once. verifyOneBlob assumes it is only called once this check succeeds.
	if err := batch.verifyKzg(scs...); err != nil {
		return nil, err
	}
	vs := make([]blocks.VerifiedROBlob, len(scs))
	for i := range scs {
		vb, err := batch.verifyOneBlob(scs[i])
		if err != nil {
			return nil, err
		}
		vs[i] = vb
	}
	return vs, nil
}

func (batch *BlobBatchVerifier) verifyOneBlob(sc blocks.ROBlob) (blocks.VerifiedROBlob, error) {
	vb := blocks.VerifiedROBlob{}
	bv := batch.newVerifier(sc, batch.reqs)
	// We can satisfy the following 2 requirements immediately because VerifiedROBlobs always verifies commitments
	// and block signature for all blobs in the batch before calling verifyOneBlob.
	bv.SatisfyRequirement(RequireSidecarKzgProofVerified)
	bv.SatisfyRequirement(RequireValidProposerSignature)

	if err := bv.BlobIndexInBounds(); err != nil {
		return vb, err
	}
	if err := bv.SidecarInclusionProven(); err != nil {
		return vb, err
	}

	return bv.VerifiedROBlob()
}

// NewDataColumnBatchVerifier initializes a data column batch verifier. It requires the caller to correctly specify
// verification Requirements and to also pass in a NewColumnVerifier, which is a callback function that
// returns a new ColumnVerifier for handling a single column in the batch.
func NewDataColumnBatchVerifier(newVerifier NewDataColumnsVerifier, reqs []Requirement) *DataColumnBatchVerifier {
	return &DataColumnBatchVerifier{
		verifyKzg:   peerdas.VerifyDataColumnsSidecarKZGProofs,
		newVerifier: newVerifier,
		reqs:        reqs,
	}
}

// DataColumnBatchVerifier solves problems that come from verifying batches of data columns from RPC.
// First: we only update forkchoice after the entire batch has completed, so the n+1 elements in the batch
// won't be in forkchoice yet.
// Second: it is more efficient to batch some verifications, like kzg commitment verification. Batch adds a
// method to ColumnVerifier to verify the kzg commitments of all data column sidecars for a block together, then using the cached
// result of the batch verification when verifying the individual columns.
type DataColumnBatchVerifier struct {
	verifyKzg   rodataColumnsCommitmentVerifier
	newVerifier NewDataColumnsVerifier
	reqs        []Requirement
}

// VerifiedRODataColumns satisfies the das.ColumnBatchVerifier interface, used by das.AvailabilityStore.
func (batch *DataColumnBatchVerifier) VerifiedRODataColumns(
	_ context.Context,
	block blocks.ROBlock,
	roDataColumns []blocks.RODataColumn,
) ([]blocks.VerifiedRODataColumn, error) {
	if len(roDataColumns) == 0 {
		return nil, nil
	}

	blockSignature := block.Signature()

	// We assume the proposer is validated wrt. the block in batch block processing before performing the DA check.
	// So at this stage we just need to make sure the value being signed and signature bytes match the block.
	for _, roDataColumn := range roDataColumns {
		dataColumnSignature := bytesutil.ToBytes96(roDataColumn.SignedBlockHeader.Signature)
		if blockSignature != dataColumnSignature {
			return nil, ErrBatchSignatureMismatch
		}

		// Extra defensive check to make sure the roots match. This should be unnecessary in practice since the root from
		// the block should be used as the lookup key into the cache of sidecars.
		if block.Root() != roDataColumn.BlockRoot() {
			return nil, ErrBatchBlockRootMismatch
		}
	}

	// Verify commitments for all columns at once. verifyOneColumn assumes it is only called once this check succeeds.
	verified, err := batch.verifyKzg(roDataColumns)
	if err != nil {
		return nil, errors.Wrap(err, "verify KZG")
	}

	if !verified {
		return nil, ErrSidecarKzgProofInvalid
	}

	dataColumnsVerifier := batch.newVerifier(roDataColumns, batch.reqs)
	// We can satisfy the following 2 requirements immediately because VerifiedROColumns always verifies commitments
	// and block signature for all columns in the batch before calling verifyOneColumn.
	dataColumnsVerifier.SatisfyRequirement(RequireSidecarKzgProofVerified)
	dataColumnsVerifier.SatisfyRequirement(RequireValidProposerSignature)

	if err := dataColumnsVerifier.DataColumnsIndexInBounds(); err != nil {
		return nil, errors.Wrap(err, "data columns index in bounds")
	}

	if err := dataColumnsVerifier.SidecarInclusionProven(); err != nil {
		return nil, errors.Wrap(err, "sidecar inclusion proved")
	}

	verifiedRODataColumns, err := dataColumnsVerifier.VerifiedRODataColumns()
	if err != nil {
		return nil, errors.Wrap(err, "verified RO data columns")
	}

	return verifiedRODataColumns, nil
}
