package blocks

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

var errNilBlockHeader = errors.New("received nil beacon block header")

type ROBlob struct {
	*ethpb.BlobSidecar
	root [32]byte
}

func NewROBlobWithRoot(b *ethpb.BlobSidecar, root [32]byte) (ROBlob, error) {
	if b == nil {
		return ROBlob{}, errNilBlock
	}
	return ROBlob{BlobSidecar: b, root: root}, nil
}

func NewROBlob(b *ethpb.BlobSidecar) (ROBlob, error) {
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
	return ROBlob{BlobSidecar: b, root: root}, nil
}
