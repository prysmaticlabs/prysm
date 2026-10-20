package interfaces

import (
	field_params "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"google.golang.org/protobuf/proto"
)

type ROSignedExecutionPayloadBid interface {
	Proto() proto.Message
	HashTreeRoot() ([32]byte, error)
	Bid() (ROExecutionPayloadBid, error)
	Signature() [field_params.BLSSignatureLength]byte
	SigningRoot([]byte) ([32]byte, error)
	IsNil() bool
}

type ROExecutionPayloadBid interface {
	ParentBlockHash() [32]byte
	ParentBlockRoot() [32]byte
	PrevRandao() [32]byte
	BlockHash() [32]byte
	GasLimit() uint64
	BuilderIndex() primitives.BuilderIndex
	Slot() primitives.Slot
	Value() primitives.Gwei
	ExecutionPayment() primitives.Gwei
	BlobKzgCommitments() [][]byte
	BlobKzgCommitmentCount() uint64
	FeeRecipient() [20]byte
	ExecutionRequestsRoot() [32]byte
	PayloadChunksRoot() [32]byte
	IsNil() bool
}
