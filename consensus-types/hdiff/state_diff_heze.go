package hdiff

import (
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
)

// executionPayloadBidSSZ returns the fork-specific latest execution payload bid of the diff.
func (s *stateDiff) executionPayloadBidSSZ() interface {
	SizeSSZ() int
	MarshalSSZTo([]byte) ([]byte, error)
} {
	if s.targetVersion >= version.Heze {
		return s.latestExecutionPayloadBidHeze
	}
	return s.latestExecutionPayloadBid
}

// hezeExecutionPayloadBid copies a Heze target's latest execution payload bid into a protobuf.
func hezeExecutionPayloadBid(bid interfaces.ROExecutionPayloadBid) *ethpb.ExecutionPayloadBidHeze {
	parentBlockHash := bid.ParentBlockHash()
	parentBlockRoot := bid.ParentBlockRoot()
	blockHash := bid.BlockHash()
	prevRandao := bid.PrevRandao()
	feeRecipient := bid.FeeRecipient()
	executionRequestsRoot := bid.ExecutionRequestsRoot()
	return &ethpb.ExecutionPayloadBidHeze{
		ParentBlockHash:       parentBlockHash[:],
		ParentBlockRoot:       parentBlockRoot[:],
		BlockHash:             blockHash[:],
		PrevRandao:            prevRandao[:],
		GasLimit:              bid.GasLimit(),
		BuilderIndex:          bid.BuilderIndex(),
		Slot:                  bid.Slot(),
		Value:                 bid.Value(),
		ExecutionPayment:      bid.ExecutionPayment(),
		BlobKzgCommitments:    bid.BlobKzgCommitments(),
		FeeRecipient:          feeRecipient[:],
		ExecutionRequestsRoot: executionRequestsRoot[:],
		InclusionListBits:     bid.InclusionListBits(),
	}
}
