package testing

import (
	"bytes"
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/statefeed"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/sirupsen/logrus"
)

// ChainService defines the mock interface for testing
type ChainService struct {
	State               *pb.BeaconState
	Root                []byte
	Block               *ethpb.BeaconBlock
	FinalizedCheckPoint *ethpb.Checkpoint
	BlocksReceived      []*ethpb.BeaconBlock
	Genesis             time.Time
	Fork                *pb.Fork
	DB                  db.Database
	stateNotifier       statefeed.Notifier
}

// StateNotifier mocks the same method in the chain service.
func (ms *ChainService) StateNotifier() statefeed.Notifier {
	if ms.stateNotifier == nil {
		ms.stateNotifier = &MockStateNotifier{}
	}
	return ms.stateNotifier
}

// MockStateNotifier mocks the state notifier.
type MockStateNotifier struct {
	feed *event.Feed
}

// StateFeed returns a state feed.
func (msn *MockStateNotifier) StateFeed() *event.Feed {
	if msn.feed == nil {
		msn.feed = new(event.Feed)
	}
	return msn.feed
}

// ReceiveBlock mocks ReceiveBlock method in chain service.
func (ms *ChainService) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

// ReceiveBlockNoVerify mocks ReceiveBlockNoVerify method in chain service.
func (ms *ChainService) ReceiveBlockNoVerify(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

// ReceiveBlockNoPubsub mocks ReceiveBlockNoPubsub method in chain service.
func (ms *ChainService) ReceiveBlockNoPubsub(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

// ReceiveBlockNoPubsubForkchoice mocks ReceiveBlockNoPubsubForkchoice method in chain service.
func (ms *ChainService) ReceiveBlockNoPubsubForkchoice(ctx context.Context, block *ethpb.BeaconBlock) error {
	if ms.State == nil {
		ms.State = &pb.BeaconState{}
	}
	if !bytes.Equal(ms.Root, block.ParentRoot) {
		return errors.Errorf("wanted %#x but got %#x", ms.Root, block.ParentRoot)
	}
	ms.State.Slot = block.Slot
	ms.BlocksReceived = append(ms.BlocksReceived, block)
	signingRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return err
	}
	if ms.DB != nil {
		if err := ms.DB.SaveBlock(ctx, block); err != nil {
			return err
		}
		logrus.Infof("Saved block with root: %#x at slot %d", signingRoot, block.Slot)
	}
	ms.Root = signingRoot[:]
	ms.Block = block
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
func (ms *ChainService) HeadState(context.Context) (*pb.BeaconState, error) {
	return ms.State, nil
}

// CurrentFork mocks HeadState method in chain service.
func (ms *ChainService) CurrentFork() *pb.Fork {
	return ms.Fork
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
	return ms.Genesis
}
