package blocks

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

var errNilBlockHeader = errors.New("received nil beacon block header")

// ROBlob represents a read-only blob sidecar with its block root.
type ROBlob struct {
	*ethpb.BlobSidecar
	root [32]byte
}

// NewROBlobWithRoot creates a new ROBlob with a given root.
func NewROBlobWithRoot(b *ethpb.BlobSidecar, root [32]byte) (ROBlob, error) {
	if b == nil {
		return ROBlob{}, errNilBlock
	}
	return ROBlob{BlobSidecar: b, root: root}, nil
}

// NewROBlob creates a new ROBlob by computing the HashTreeRoot of the header.
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

// BlockRoot returns the root of the block.
func (b *ROBlob) BlockRoot() [32]byte {
	return b.root
}

// Slot returns the slot of the blob sidecar.
func (b *ROBlob) Slot() primitives.Slot {
	return b.SignedBlockHeader.Header.Slot
}

// ParentRoot returns the parent root of the blob sidecar.
func (b *ROBlob) ParentRoot() [32]byte {
	return bytesutil.ToBytes32(b.SignedBlockHeader.Header.ParentRoot)
}

// BodyRoot returns the body root of the blob sidecar.
func (b *ROBlob) BodyRoot() [32]byte {
	return bytesutil.ToBytes32(b.SignedBlockHeader.Header.BodyRoot)
}

// ProposerIndex returns the proposer index of the blob sidecar.
func (b *ROBlob) ProposerIndex() primitives.ValidatorIndex {
	return b.SignedBlockHeader.Header.ProposerIndex
}

func (b *ROBlob) BlockRootSlice() []byte {
	return b.root[:]
}

type ROBlobSlice []ROBlob

func (s ROBlobSlice) Protos() []*ethpb.BlobSidecar {
	pb := make([]*ethpb.BlobSidecar, len(s))
	for i := range s {
		pb[i] = s[i].BlobSidecar
	}
	return pb
}

type VerifiedROBlob struct {
	ROBlob
}

func NewVerifiedBlobSlice(pbs []*ethpb.BlobSidecar, root [32]byte) ([]VerifiedROBlob, error) {
	vs := make([]VerifiedROBlob, len(pbs))
	var err error
	for i := range pbs {
		vs[i], err = NewVerifiedBlobWithRoot(pbs[i], root)
		if err != nil {
			return nil, err
		}
	}
	return vs, nil
}

func NewVerifiedBlobWithRoot(pb *ethpb.BlobSidecar, root [32]byte) (VerifiedROBlob, error) {
	r, err := NewROBlobWithRoot(pb, root)
	if err != nil {
		return VerifiedROBlob{}, err
	}
	return VerifiedROBlob{ROBlob: r}, nil
}

type VerifiedROBlobSlice []VerifiedROBlob

func (s VerifiedROBlobSlice) ROBlobs() []ROBlob {
	robs := make([]ROBlob, len(s))
	for i := range s {
		robs[i] = s[i].ROBlob
	}
	return robs
}

func (s VerifiedROBlobSlice) Protos() []*ethpb.BlobSidecar {
	pbs := make([]*ethpb.BlobSidecar, len(s))
	for i := range s {
		pbs[i] = s[i].BlobSidecar
	}
	return pbs
}
