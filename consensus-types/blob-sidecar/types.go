package blob_sidecar

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

var ErrUnsupportedBlobSidecar = errors.New("unsupported blob sidecar")

type BlobSidecar struct {
	slot              primitives.Slot
	index             uint64
	blockRoot         []byte
	parentBlockRoot   []byte
	proposerIndex     primitives.ValidatorIndex
	blob              []byte
	kzgCommitment     []byte
	kzgProof          []byte
	kzgInclusionProof [][]byte
}

func New(i interface{}) (interfaces.BlobSidecar, error) {
	switch b := i.(type) {
	case nil:
		return nil, blocks.ErrNilObject
	case *ethpb.BlobSidecar:
		return &BlobSidecar{
			slot:              b.Slot,
			index:             b.Index,
			blockRoot:         b.BlockRoot,
			parentBlockRoot:   b.BlockParentRoot,
			proposerIndex:     b.ProposerIndex,
			blob:              b.Blob,
			kzgCommitment:     b.KzgCommitment,
			kzgProof:          b.KzgProof,
			kzgInclusionProof: [][]byte{}, // TODO: kzg inclusion proof is not available at the older version of blob sidecar. This version will be deleted very shortly anyway.
		}, nil
	case *ethpb.BlobSidecarNew:
		header := b.SignedBlockHeader.Header
		blockRoot, err := header.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		return &BlobSidecar{
			slot:              header.Slot,
			index:             b.Index,
			blockRoot:         blockRoot[:],
			parentBlockRoot:   header.ParentRoot,
			proposerIndex:     header.ProposerIndex,
			blob:              b.Blob,
			kzgCommitment:     b.KzgCommitment,
			kzgProof:          b.KzgProof,
			kzgInclusionProof: b.CommitmentInclusionProof,
		}, nil
	default:
		return nil, errors.Wrapf(ErrUnsupportedBlobSidecar, "got %T", b)
	}
}
