package blockchain

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ChainInfoFetcher defines a common interface for methods in blockchain service which
// directly retrieves chain info related data.
type ChainInfoFetcher interface {
	HeadFetcher
	CanonicalRootFetcher
	FinalizationFetcher
}

// GenesisTimeFetcher retrieves the Eth2 genesis timestamp.
type GenesisTimeFetcher interface {
	GenesisTime() time.Time
}

// HeadFetcher defines a common interface for methods in blockchain service which
// directly retrieves head related data.
type HeadFetcher interface {
	HeadSlot() uint64
	HeadRoot() []byte
	HeadBlock() *ethpb.BeaconBlock
	HeadState(ctx context.Context) (*pb.BeaconState, error)
}

// CanonicalRootFetcher defines a common interface for methods in blockchain service which
// directly retrieves canonical roots related data.
type CanonicalRootFetcher interface {
	CanonicalRoot(slot uint64) []byte
}

// ForkFetcher retrieves the current fork information of the Ethereum beacon chain.
type ForkFetcher interface {
	CurrentFork() *pb.Fork
}

// FinalizationFetcher defines a common interface for methods in blockchain service which
// directly retrieves finalization related data.
type FinalizationFetcher interface {
	FinalizedCheckpt() *ethpb.Checkpoint
}

// FinalizedCheckpt returns the latest finalized checkpoint tracked in fork choice service.
func (s *Service) FinalizedCheckpt() *ethpb.Checkpoint {
	cp := s.forkChoiceStore.FinalizedCheckpt()
	if cp != nil {
		return cp
	}

	return &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
}

// HeadSlot returns the slot of the head of the chain.
func (s *Service) HeadSlot() uint64 {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.headSlot
}

// HeadRoot returns the root of the head of the chain.
func (s *Service) HeadRoot() []byte {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	root := s.canonicalRoots[s.headSlot]
	if len(root) != 0 {
		return root
	}

	return params.BeaconConfig().ZeroHash[:]
}

// HeadBlock returns the head block of the chain.
func (s *Service) HeadBlock() *ethpb.BeaconBlock {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return proto.Clone(s.headBlock).(*ethpb.BeaconBlock)
}

// HeadState returns the head state of the chain.
// If the head state is nil from service struct,
// it will attempt to get from DB and error if nil again.
func (s *Service) HeadState(ctx context.Context) (*pb.BeaconState, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if s.headState == nil {
		return s.beaconDB.HeadState(ctx)
	}

	return proto.Clone(s.headState).(*pb.BeaconState), nil
}

// CanonicalRoot returns the canonical root of a given slot.
func (s *Service) CanonicalRoot(slot uint64) []byte {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.canonicalRoots[slot]
}

// GenesisTime returns the genesis time of beacon chain.
func (s *Service) GenesisTime() time.Time {
	return s.genesisTime
}

// CurrentFork retrieves the latest fork information of the beacon chain.
func (s *Service) CurrentFork() *pb.Fork {
	if s.headState == nil {
		return &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		}
	}
	return proto.Clone(s.headState.Fork).(*pb.Fork)
}
