package interfaces

import "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"

type BlobSidecar interface {
	Slot() primitives.Slot
	Index() uint64
	BlockRoot() []byte
	ParentBlockRoot() []byte
	ProposerIndex() primitives.ValidatorIndex
	Blob() []byte
	KzgCommitment() []byte
	KzgProof() []byte
	KzgInclusionProof() [][]byte
}
