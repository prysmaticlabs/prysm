// Package testing includes useful mocks for writing unit
// tests which depend on logic from the blockchain package.
package testing

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	opfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

// ChainService defines the mock interface for testing
type ChainService struct {
	State                       *stateTrie.BeaconState
	Root                        []byte
	Block                       *ethpb.SignedBeaconBlock
	FinalizedCheckPoint         *ethpb.Checkpoint
	CurrentJustifiedCheckPoint  *ethpb.Checkpoint
	PreviousJustifiedCheckPoint *ethpb.Checkpoint
	BlocksReceived              []*ethpb.SignedBeaconBlock
	Balance                     *precompute.Balance
	Genesis                     time.Time
	ValidatorsRoot              [32]byte
	Fork                        *pb.Fork
	ETH1Data                    *ethpb.Eth1Data
	DB                          db.Database
	stateNotifier               statefeed.Notifier
	blockNotifier               blockfeed.Notifier
	opNotifier                  opfeed.Notifier
	ValidAttestation            bool
	ForkChoiceStore             *protoarray.Store
}

// StateNotifier mocks the same method in the chain service.
func (ms *ChainService) StateNotifier() statefeed.Notifier {
	if ms.stateNotifier == nil {
		ms.stateNotifier = &MockStateNotifier{}
	}
	return ms.stateNotifier
}

// BlockNotifier mocks the same method in the chain service.
func (ms *ChainService) BlockNotifier() blockfeed.Notifier {
	if ms.blockNotifier == nil {
		ms.blockNotifier = &MockBlockNotifier{}
	}
	return ms.blockNotifier
}

// MockBlockNotifier mocks the block notifier.
type MockBlockNotifier struct {
	feed *event.Feed
}

// BlockFeed returns a block feed.
func (msn *MockBlockNotifier) BlockFeed() *event.Feed {
	if msn.feed == nil {
		msn.feed = new(event.Feed)
	}
	return msn.feed
}

// MockStateNotifier mocks the state notifier.
type MockStateNotifier struct {
	feed *event.Feed

	recv []*feed.Event
	recvLock sync.Mutex
	recvCh chan *feed.Event
}

func (msn *MockStateNotifier) ReceivedEvents() []*feed.Event {
	msn.recvLock.Lock()
	defer msn.recvLock.Unlock()
	return msn.recv
}

// StateFeed returns a state feed.
func (msn *MockStateNotifier) StateFeed() *event.Feed {
	if msn.feed == nil && msn.recvCh == nil {
		msn.feed = new(event.Feed)
		msn.recvCh = make(chan *feed.Event)
		sub := msn.feed.Subscribe(msn.recvCh)

		go func() {
			select {
			case evt := <-msn.recvCh:
				msn.recvLock.Lock()
				msn.recv = append(msn.recv, evt)
				msn.recvLock.Unlock()
			case <-sub.Err():
				sub.Unsubscribe()
			}
		}()
	}
	return msn.feed
}

// OperationNotifier mocks the same method in the chain service.
func (ms *ChainService) OperationNotifier() opfeed.Notifier {
	if ms.opNotifier == nil {
		ms.opNotifier = &MockOperationNotifier{}
	}
	return ms.opNotifier
}

// MockOperationNotifier mocks the operation notifier.
type MockOperationNotifier struct {
	feed *event.Feed
}

// OperationFeed returns an operation feed.
func (mon *MockOperationNotifier) OperationFeed() *event.Feed {
	if mon.feed == nil {
		mon.feed = new(event.Feed)
	}
	return mon.feed
}

// ReceiveBlock mocks ReceiveBlock method in chain service.
func (ms *ChainService) ReceiveBlock(ctx context.Context, block *ethpb.SignedBeaconBlock, blockRoot [32]byte) error {
	return nil
}

// ReceiveBlockInitialSync mocks ReceiveBlockInitialSync method in chain service.
func (ms *ChainService) ReceiveBlockInitialSync(ctx context.Context, block *ethpb.SignedBeaconBlock, blockRoot [32]byte) error {
	if ms.State == nil {
		ms.State = &stateTrie.BeaconState{}
	}
	if !bytes.Equal(ms.Root, block.Block.ParentRoot) {
		return errors.Errorf("wanted %#x but got %#x", ms.Root, block.Block.ParentRoot)
	}
	if err := ms.State.SetSlot(block.Block.Slot); err != nil {
		return err
	}
	ms.BlocksReceived = append(ms.BlocksReceived, block)
	signingRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		return err
	}
	if ms.DB != nil {
		if err := ms.DB.SaveBlock(ctx, block); err != nil {
			return err
		}
		logrus.Infof("Saved block with root: %#x at slot %d", signingRoot, block.Block.Slot)
	}
	ms.Root = signingRoot[:]
	ms.Block = block
	return nil
}

// ReceiveBlockNoPubsub mocks ReceiveBlockNoPubsub method in chain service.
func (ms *ChainService) ReceiveBlockNoPubsub(ctx context.Context, block *ethpb.SignedBeaconBlock, blockRoot [32]byte) error {
	if ms.State == nil {
		ms.State = &stateTrie.BeaconState{}
	}
	if !bytes.Equal(ms.Root, block.Block.ParentRoot) {
		return errors.Errorf("wanted %#x but got %#x", ms.Root, block.Block.ParentRoot)
	}
	if err := ms.State.SetSlot(block.Block.Slot); err != nil {
		return err
	}
	ms.BlocksReceived = append(ms.BlocksReceived, block)
	signingRoot, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		return err
	}
	if ms.DB != nil {
		if err := ms.DB.SaveBlock(ctx, block); err != nil {
			return err
		}
		logrus.Infof("Saved block with root: %#x at slot %d", signingRoot, block.Block.Slot)
	}
	ms.Root = signingRoot[:]
	ms.Block = block
	return nil
}

// HeadSlot mocks HeadSlot method in chain service.
func (ms *ChainService) HeadSlot() uint64 {
	if ms.State == nil {
		return 0
	}
	return ms.State.Slot()
}

// HeadRoot mocks HeadRoot method in chain service.
func (ms *ChainService) HeadRoot(ctx context.Context) ([]byte, error) {
	return ms.Root, nil

}

// HeadBlock mocks HeadBlock method in chain service.
func (ms *ChainService) HeadBlock(context.Context) (*ethpb.SignedBeaconBlock, error) {
	return ms.Block, nil
}

// HeadState mocks HeadState method in chain service.
func (ms *ChainService) HeadState(context.Context) (*stateTrie.BeaconState, error) {
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

// CurrentJustifiedCheckpt mocks CurrentJustifiedCheckpt method in chain service.
func (ms *ChainService) CurrentJustifiedCheckpt() *ethpb.Checkpoint {
	return ms.CurrentJustifiedCheckPoint
}

// PreviousJustifiedCheckpt mocks PreviousJustifiedCheckpt method in chain service.
func (ms *ChainService) PreviousJustifiedCheckpt() *ethpb.Checkpoint {
	return ms.PreviousJustifiedCheckPoint
}

// ReceiveAttestation mocks ReceiveAttestation method in chain service.
func (ms *ChainService) ReceiveAttestation(context.Context, *ethpb.Attestation) error {
	return nil
}

// ReceiveAttestationNoPubsub mocks ReceiveAttestationNoPubsub method in chain service.
func (ms *ChainService) ReceiveAttestationNoPubsub(context.Context, *ethpb.Attestation) error {
	return nil
}

// AttestationPreState mocks AttestationPreState method in chain service.
func (ms *ChainService) AttestationPreState(ctx context.Context, att *ethpb.Attestation) (*stateTrie.BeaconState, error) {
	return ms.State, nil
}

// HeadValidatorsIndices mocks the same method in the chain service.
func (ms *ChainService) HeadValidatorsIndices(ctx context.Context, epoch uint64) ([]uint64, error) {
	if ms.State == nil {
		return []uint64{}, nil
	}
	return helpers.ActiveValidatorIndices(ms.State, epoch)
}

// HeadSeed mocks the same method in the chain service.
func (ms *ChainService) HeadSeed(ctx context.Context, epoch uint64) ([32]byte, error) {
	return helpers.Seed(ms.State, epoch, params.BeaconConfig().DomainBeaconAttester)
}

// HeadETH1Data provides the current ETH1Data of the head state.
func (ms *ChainService) HeadETH1Data() *ethpb.Eth1Data {
	return ms.ETH1Data
}

// ProtoArrayStore mocks the same method in the chain service.
func (ms *ChainService) ProtoArrayStore() *protoarray.Store {
	return ms.ForkChoiceStore
}

// GenesisTime mocks the same method in the chain service.
func (ms *ChainService) GenesisTime() time.Time {
	return ms.Genesis
}

// GenesisValidatorRoot mocks the same method in the chain service.
func (ms *ChainService) GenesisValidatorRoot() [32]byte {
	return ms.ValidatorsRoot
}

// CurrentSlot mocks the same method in the chain service.
func (ms *ChainService) CurrentSlot() uint64 {
	return uint64(time.Now().Unix()-ms.Genesis.Unix()) / params.BeaconConfig().SecondsPerSlot
}

// Participation mocks the same method in the chain service.
func (ms *ChainService) Participation(epoch uint64) *precompute.Balance {
	return ms.Balance
}

// IsValidAttestation always returns true.
func (ms *ChainService) IsValidAttestation(ctx context.Context, att *ethpb.Attestation) bool {
	return ms.ValidAttestation
}

// IsCanonical returns and determines whether a block with the provided root is part of
// the canonical chain.
func (ms *ChainService) IsCanonical(ctx context.Context, blockRoot [32]byte) (bool, error) {
	return true, nil
}

// ClearCachedStates does nothing.
func (ms *ChainService) ClearCachedStates() {}

// HasInitSyncBlock mocks the same method in the chain service.
func (ms *ChainService) HasInitSyncBlock(root [32]byte) bool {
	return false
}

// HeadGenesisValidatorRoot mocks HeadGenesisValidatorRoot method in chain service.
func (ms *ChainService) HeadGenesisValidatorRoot() [32]byte {
	return [32]byte{}
}
