package testing

import (
	"context"
	"time"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
)

// ChainService defines the mock interface for testing
type ChainService struct {
	State               *pb.BeaconState
	Root                []byte
	Block               *ethpb.BeaconBlock
	FinalizedCheckPoint *ethpb.Checkpoint
	StateFeed           *event.Feed
}

// ReceiveBlock mocks ReceiveBlock method in chain service.
func (ms *ChainService) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

// ReceiveBlockNoPubsub mocks ReceiveBlockNoPubsub method in chain service.
func (ms *ChainService) ReceiveBlockNoPubsub(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

// ReceiveBlockNoPubsubForkchoice mocks ReceiveBlockNoPubsubForkchoice method in chain service.
func (ms *ChainService) ReceiveBlockNoPubsubForkchoice(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

// HeadSlot mocks HeadSlot method in chain service.
func (ms *ChainService) HeadSlot() uint64 {
	return ms.State.Slot

}

// HeadRoot mocks HeadRoot method in chain service.
func (ms *ChainService) HeadRoot() []byte {
	return ms.Root

}

// HeadBlock mocks HeadBlock method in chain service.
func (ms *ChainService) HeadBlock() *ethpb.BeaconBlock {
	return ms.Block
}

// HeadState mocks HeadState method in chain service.
func (ms *ChainService) HeadState() *pb.BeaconState {
	return ms.State
}

// FinalizedCheckpt mocks FinalizedCheckpt method in chain service.
func (ms *ChainService) FinalizedCheckpt() *ethpb.Checkpoint {
	return ms.FinalizedCheckPoint
}

// ReceiveAttestation mocks ReceiveAttestation method in chain service.
func (ms *ChainService) ReceiveAttestation(context.Context, *ethpb.Attestation) error {
	return nil
}

// ReceiveAttestationNoPubsub mocks ReceiveAttestationNoPubsub method in chain service.
func (ms *ChainService) ReceiveAttestationNoPubsub(context.Context, *ethpb.Attestation) error {
	return nil
}

// GenesisTime mocks the same method in the chain service.
func (ms *ChainService) GenesisTime() time.Time {
	return time.Unix(0, 0)
}

// StateInitializedFeed mocks the same method in the chain service.
func (ms *ChainService) StateInitializedFeed() *event.Feed {
	if ms.StateFeed != nil {
		return ms.StateFeed
	}
	ms.StateFeed = new(event.Feed)
	return ms.StateFeed
}

// HeadUpdatedFeed mocks the same method in the chain service.
func (ms *ChainService) HeadUpdatedFeed() *event.Feed {
	return new(event.Feed)
}
