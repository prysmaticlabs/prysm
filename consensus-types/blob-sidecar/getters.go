package blob_sidecar

import "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"

func (b *BlobSidecar) Slot() primitives.Slot {
	return b.slot
}

func (b *BlobSidecar) Index() uint64 {
	return b.index
}

func (b *BlobSidecar) BlockRoot() []byte {
	return b.blockRoot
}

func (b *BlobSidecar) ParentBlockRoot() []byte {
	return b.parentBlockRoot
}

func (b *BlobSidecar) ProposerIndex() primitives.ValidatorIndex {
	return b.proposerIndex
}

func (b *BlobSidecar) Blob() []byte {
	return b.blob
}

func (b *BlobSidecar) KzgCommitment() []byte {
	return b.kzgCommitment
}

func (b *BlobSidecar) KzgProof() []byte {
	return b.kzgProof
}

func (b *BlobSidecar) KzgInclusionProof() [][]byte {
	return b.kzgInclusionProof
}
