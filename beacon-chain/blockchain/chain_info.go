package blockchain

import (
	"bytes"
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
	// Deprecated: Use beacondb.TODO.
	HeadSlot() uint64
	// Deprecated: Use beacondb.TODO.
	HeadRoot() []byte
	// Deprecated: Use beacondb.HeadBlock.
	HeadBlock() *ethpb.SignedBeaconBlock
	// Deprecated: Use beacondb.HeadState.
	HeadState(ctx context.Context) (*pb.BeaconState, error)
	// Deprecated: Use beacondb.TODO.
	HeadValidatorsIndices(epoch uint64) ([]uint64, error)
	// Deprecated: Use beacondb.TODO.
	HeadSeed(epoch uint64) ([32]byte, error)
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
// directly retrieves finalization and justification related data.
type FinalizationFetcher interface {
	// Deprecated: Use beacondb.FinalizedCheckpoint.
	FinalizedCheckpt() *ethpb.Checkpoint
	// Deprecated: Use beacondb.CurrentJustifiedCheckpoint.
	CurrentJustifiedCheckpt() *ethpb.Checkpoint
	// Deprecated: Use beacondb.TODO.
	// TODO: Write a beacondb method to return this value.
	PreviousJustifiedCheckpt() *ethpb.Checkpoint
}

// ParticipationFetcher defines a common interface for methods in blockchain service which
// directly retrieves validator participation related data.
type ParticipationFetcher interface {
	Participation(epoch uint64) *precompute.Balance
}

// FinalizedCheckpt returns the latest finalized checkpoint from head state.
// Deprecated: Use beacondb.FinalizedCheckpoint.
func (s *Service) FinalizedCheckpt() *ethpb.Checkpoint {
	headState, err := s.beaconDB.HeadState(context.TODO())
	if err != nil || headState == nil || headState.FinalizedCheckpoint == nil {
		return &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	}

	// If head state exists but there hasn't been a finalized check point,
	// the check point's root should refer to genesis block root.
	if bytes.Equal(headState.FinalizedCheckpoint.Root, params.BeaconConfig().ZeroHash[:]) {
		return &ethpb.Checkpoint{Root: s.genesisRoot[:]}
	}

	return headState.FinalizedCheckpoint
}

// CurrentJustifiedCheckpt returns the current justified checkpoint from head state.
// Deprecated: Use beacondb.JustifiedCheckpoint.
func (s *Service) CurrentJustifiedCheckpt() *ethpb.Checkpoint {
	c, _ := s.beaconDB.JustifiedCheckpoint(context.TODO())

	// If head state exists but there hasn't been a justified check point,
	// the check point root should refer to genesis block root.
	if bytes.Equal(c.Root, params.BeaconConfig().ZeroHash[:]) {
		return &ethpb.Checkpoint{Root: s.genesisRoot[:]}
	}
	return c
}

// PreviousJustifiedCheckpt returns the previous justified checkpoint from head state.
// TODO: This is a full state read that could be another method in the DB.
func (s *Service) PreviousJustifiedCheckpt() *ethpb.Checkpoint {
	headState, _ := s.HeadState(context.TODO())
	if headState == nil || headState.PreviousJustifiedCheckpoint == nil {
		return &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	}

	// If head state exists but there hasn't been a justified check point,
	// the check point root should refer to genesis block root.
	if bytes.Equal(headState.PreviousJustifiedCheckpoint.Root, params.BeaconConfig().ZeroHash[:]) {
		return &ethpb.Checkpoint{Root: s.genesisRoot[:]}
	}

	return headState.PreviousJustifiedCheckpoint
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
// Deprecated: Use beacondb.HeadBlock.
func (s *Service) HeadBlock() *ethpb.SignedBeaconBlock {
	hb, _ := s.beaconDB.HeadBlock(context.TODO())
	return hb
}

// HeadState returns the head state of the chain.
// If the head state is nil from service struct,
// it will attempt to get from DB and error if nil again.
// Deprecated: Use beacondb.HeadState.
func (s *Service) HeadState(ctx context.Context) (*pb.BeaconState, error) {
	return s.beaconDB.HeadState(ctx)
}

// HeadValidatorsIndices returns a list of active validator indices from the head view of a given epoch.
// TODO: This is a full state read. Maybe it can be reduced?
func (s *Service) HeadValidatorsIndices(epoch uint64) ([]uint64, error) {
	headState, _ := s.HeadState(context.TODO())
	if headState == nil {
		return []uint64{}, nil
	}
	return helpers.ActiveValidatorIndices(headState, epoch)
}

// HeadSeed returns the seed from the head view of a given epoch.
func (s *Service) HeadSeed(epoch uint64) ([32]byte, error) {
	headState, err := s.beaconDB.HeadState(context.TODO())
	if err != nil {
		return [32]byte{}, err
	}
	if headState == nil {
		return [32]byte{}, nil
	}

	return helpers.Seed(headState, epoch, params.BeaconConfig().DomainBeaconAttester)
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
// TODO: Can this be improved? It is a full state read now just for the fork.
func (s *Service) CurrentFork() *pb.Fork {
	headState, _ := s.beaconDB.HeadState(context.TODO())
	if headState == nil {
		return &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		}
	}
	return proto.Clone(headState.Fork).(*pb.Fork)
}

// Participation returns the participation stats of a given epoch.
func (s *Service) Participation(epoch uint64) *precompute.Balance {
	s.epochParticipationLock.RLock()
	defer s.epochParticipationLock.RUnlock()

	return s.epochParticipation[epoch]
}
