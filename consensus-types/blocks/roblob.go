package blocks

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

var errNilBlockHeader = errors.New("received nil beacon block header")

type ROBlob struct {
	*ethpb.BlobSidecarNew
	root [32]byte
}

func NewROBlobWithRoot(b *ethpb.BlobSidecarNew, root [32]byte) (ROBlob, error) {
	if b == nil {
		return ROBlob{}, errNilBlock
	}
	return ROBlob{BlobSidecarNew: b, root: root}, nil
}

func NewROBlob(b *ethpb.BlobSidecarNew) (ROBlob, error) {
	if b == nil {
		return ROBlob{}, errNilBlock
	}
	if b.SignedBlockHeader == nil || b.SignedBlockHeader.Header == nil {
		return ROBlob{}, errNilBlockHeader
	}
	root, err := b.SignedBlockHeader.Header.HashTreeRoot()
	if err != nil {
		return ROBlob{}, err
	}
	return ROBlob{BlobSidecarNew: b, root: root}, nil
}
