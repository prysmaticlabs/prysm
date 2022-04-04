package blockchain

import (
	"context"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	doublylinkedtree "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/time/slots"
	"go.opencensus.io/trace"
)

// ChainInfoFetcher defines a common interface for methods in blockchain service which
// directly retrieves chain info related data.
type ChainInfoFetcher interface {
	HeadFetcher
	FinalizationFetcher
	GenesisFetcher
	CanonicalFetcher
	ForkFetcher
	TimeFetcher
	HeadDomainFetcher
}

// TimeFetcher retrieves the Ethereum consensus data that's related to time.
type TimeFetcher interface {
	GenesisTime() time.Time
	CurrentSlot() types.Slot
}

// GenesisFetcher retrieves the Ethereum consensus data related to its genesis.
type GenesisFetcher interface {
	GenesisValidatorsRoot() [32]byte
}

// HeadFetcher defines a common interface for methods in blockchain service which
// directly retrieves head related data.
type HeadFetcher interface {
	HeadSlot() types.Slot
	HeadRoot(ctx context.Context) ([]byte, error)
	HeadBlock(ctx context.Context) (block.SignedBeaconBlock, error)
	HeadState(ctx context.Context) (state.BeaconState, error)
	HeadValidatorsIndices(ctx context.Context, epoch types.Epoch) ([]types.ValidatorIndex, error)
	HeadSeed(ctx context.Context, epoch types.Epoch) ([32]byte, error)
	HeadGenesisValidatorsRoot() [32]byte
	HeadETH1Data() *ethpb.Eth1Data
	HeadPublicKeyToValidatorIndex(pubKey [fieldparams.BLSPubkeyLength]byte) (types.ValidatorIndex, bool)
	HeadValidatorIndexToPublicKey(ctx context.Context, index types.ValidatorIndex) ([fieldparams.BLSPubkeyLength]byte, error)
	ChainHeads() ([][32]byte, []types.Slot)
	IsOptimistic(ctx context.Context) (bool, error)
	IsOptimisticForRoot(ctx context.Context, root [32]byte) (bool, error)
	HeadSyncCommitteeFetcher
	HeadDomainFetcher
	ForkChoicer() forkchoice.ForkChoicer
}

// ForkFetcher retrieves the current fork information of the Ethereum beacon chain.
type ForkFetcher interface {
	CurrentFork() *ethpb.Fork
}

// CanonicalFetcher retrieves the current chain's canonical information.
type CanonicalFetcher interface {
	IsCanonical(ctx context.Context, blockRoot [32]byte) (bool, error)
	VerifyBlkDescendant(ctx context.Context, blockRoot [32]byte) error
}

// FinalizationFetcher defines a common interface for methods in blockchain service which
// directly retrieves finalization and justification related data.
type FinalizationFetcher interface {
	FinalizedCheckpt() *ethpb.Checkpoint
	CurrentJustifiedCheckpt() *ethpb.Checkpoint
	PreviousJustifiedCheckpt() *ethpb.Checkpoint
}

// FinalizedCheckpt returns the latest finalized checkpoint from chain store.
func (s *Service) FinalizedCheckpt() *ethpb.Checkpoint {
	cp := s.store.FinalizedCheckpt()
	if cp == nil {
		return &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	}

	return ethpb.CopyCheckpoint(cp)
}

// CurrentJustifiedCheckpt returns the current justified checkpoint from chain store.
func (s *Service) CurrentJustifiedCheckpt() *ethpb.Checkpoint {
	cp := s.store.JustifiedCheckpt()
	if cp == nil {
		return &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	}

	return ethpb.CopyCheckpoint(cp)
}

// PreviousJustifiedCheckpt returns the previous justified checkpoint from chain store.
func (s *Service) PreviousJustifiedCheckpt() *ethpb.Checkpoint {
	cp := s.store.PrevJustifiedCheckpt()
	if cp == nil {
		return &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	}

	return ethpb.CopyCheckpoint(cp)
}

// BestJustifiedCheckpt returns the best justified checkpoint from store.
func (s *Service) BestJustifiedCheckpt() *ethpb.Checkpoint {
	cp := s.store.BestJustifiedCheckpt()
	// If there is no best justified checkpoint, return the checkpoint with root as zeros to be used for genesis cases.
	if cp == nil {
		return &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	}

	return ethpb.CopyCheckpoint(cp)
}

// HeadSlot returns the slot of the head of the chain.
func (s *Service) HeadSlot() types.Slot {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if !s.hasHeadState() {
		return 0
	}

	return s.headSlot()
}

// HeadRoot returns the root of the head of the chain.
func (s *Service) HeadRoot(ctx context.Context) ([]byte, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if s.headRoot() != params.BeaconConfig().ZeroHash {
		r := s.headRoot()
		return r[:], nil
	}

	b, err := s.cfg.BeaconDB.HeadBlock(ctx)
	if err != nil {
		return nil, err
	}
	if b == nil || b.IsNil() {
		return params.BeaconConfig().ZeroHash[:], nil
	}

	r, err := b.Block().HashTreeRoot()
	if err != nil {
		return nil, err
	}

	return r[:], nil
}

// HeadBlock returns the head block of the chain.
// If the head is nil from service struct,
// it will attempt to get the head block from DB.
func (s *Service) HeadBlock(ctx context.Context) (block.SignedBeaconBlock, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if s.hasHeadState() {
		return s.headBlock(), nil
	}

	return s.cfg.BeaconDB.HeadBlock(ctx)
}

// HeadState returns the head state of the chain.
// If the head is nil from service struct,
// it will attempt to get the head state from DB.
func (s *Service) HeadState(ctx context.Context) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.HeadState")
	defer span.End()
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	ok := s.hasHeadState()
	span.AddAttributes(trace.BoolAttribute("cache_hit", ok))

	if ok {
		return s.headState(ctx), nil
	}

	return s.cfg.StateGen.StateByRoot(ctx, s.headRoot())
}

// HeadValidatorsIndices returns a list of active validator indices from the head view of a given epoch.
func (s *Service) HeadValidatorsIndices(ctx context.Context, epoch types.Epoch) ([]types.ValidatorIndex, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if !s.hasHeadState() {
		return []types.ValidatorIndex{}, nil
	}
	return helpers.ActiveValidatorIndices(ctx, s.headState(ctx), epoch)
}

// HeadSeed returns the seed from the head view of a given epoch.
func (s *Service) HeadSeed(ctx context.Context, epoch types.Epoch) ([32]byte, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if !s.hasHeadState() {
		return [32]byte{}, nil
	}

	return helpers.Seed(s.headState(ctx), epoch, params.BeaconConfig().DomainBeaconAttester)
}

// HeadGenesisValidatorsRoot returns genesis validators root of the head state.
func (s *Service) HeadGenesisValidatorsRoot() [32]byte {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if !s.hasHeadState() {
		return [32]byte{}
	}

	return s.headGenesisValidatorsRoot()
}

// HeadETH1Data returns the eth1data of the current head state.
func (s *Service) HeadETH1Data() *ethpb.Eth1Data {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if !s.hasHeadState() {
		return &ethpb.Eth1Data{}
	}
	return s.head.state.Eth1Data()
}

// GenesisTime returns the genesis time of beacon chain.
func (s *Service) GenesisTime() time.Time {
	return s.genesisTime
}

// GenesisValidatorsRoot returns the genesis validator
// root of the chain.
func (s *Service) GenesisValidatorsRoot() [32]byte {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if !s.hasHeadState() {
		return [32]byte{}
	}
	return bytesutil.ToBytes32(s.head.state.GenesisValidatorsRoot())
}

// CurrentFork retrieves the latest fork information of the beacon chain.
func (s *Service) CurrentFork() *ethpb.Fork {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if !s.hasHeadState() {
		return &ethpb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		}
	}
	return s.head.state.Fork()
}

// IsCanonical returns true if the input block root is part of the canonical chain.
func (s *Service) IsCanonical(ctx context.Context, blockRoot [32]byte) (bool, error) {
	// If the block has been finalized, the block will always be part of the canonical chain.
	if s.cfg.BeaconDB.IsFinalizedBlock(ctx, blockRoot) {
		return true, nil
	}

	// If the block has not been finalized, check fork choice store to see if the block is canonical
	return s.cfg.ForkChoiceStore.IsCanonical(blockRoot), nil
}

// ChainHeads returns all possible chain heads (leaves of fork choice tree).
// Heads roots and heads slots are returned.
func (s *Service) ChainHeads() ([][32]byte, []types.Slot) {
	return s.cfg.ForkChoiceStore.Tips()
}

// HeadPublicKeyToValidatorIndex returns the validator index of the `pubkey` in current head state.
func (s *Service) HeadPublicKeyToValidatorIndex(pubKey [fieldparams.BLSPubkeyLength]byte) (types.ValidatorIndex, bool) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()
	if !s.hasHeadState() {
		return 0, false
	}
	return s.headValidatorIndexAtPubkey(pubKey)
}

// HeadValidatorIndexToPublicKey returns the pubkey of the validator `index`  in current head state.
func (s *Service) HeadValidatorIndexToPublicKey(_ context.Context, index types.ValidatorIndex) ([fieldparams.BLSPubkeyLength]byte, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()
	if !s.hasHeadState() {
		return [fieldparams.BLSPubkeyLength]byte{}, nil
	}
	v, err := s.headValidatorAtIndex(index)
	if err != nil {
		return [fieldparams.BLSPubkeyLength]byte{}, err
	}
	return v.PublicKey(), nil
}

// ForkChoicer returns the forkchoice interface
func (s *Service) ForkChoicer() forkchoice.ForkChoicer {
	return s.cfg.ForkChoiceStore
}

// IsOptimistic returns true if the current head is optimistic.
func (s *Service) IsOptimistic(ctx context.Context) (bool, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()
	if slots.ToEpoch(s.CurrentSlot()) < params.BeaconConfig().BellatrixForkEpoch {
		return false, nil
	}

	return s.IsOptimisticForRoot(ctx, s.head.root)
}

// IsOptimisticForRoot takes the root and slot as arguments instead of the current head
// and returns true if it is optimistic.
func (s *Service) IsOptimisticForRoot(ctx context.Context, root [32]byte) (bool, error) {
	optimistic, err := s.cfg.ForkChoiceStore.IsOptimistic(ctx, root)
	if err == nil {
		return optimistic, nil
	}
	if err != protoarray.ErrUnknownNodeRoot && err != doublylinkedtree.ErrNilNode {
		return false, err
	}
	ss, err := s.cfg.BeaconDB.StateSummary(ctx, root)
	if err != nil {
		return false, err
	}
	if ss == nil {
		return false, errInvalidNilSummary
	}

	validatedCheckpoint, err := s.cfg.BeaconDB.LastValidatedCheckpoint(ctx)
	if err != nil {
		return false, err
	}
	if slots.ToEpoch(ss.Slot) > validatedCheckpoint.Epoch {
		return true, nil
	}

	if slots.ToEpoch(ss.Slot)+1 < validatedCheckpoint.Epoch {
		return false, nil
	}

	lastValidated, err := s.cfg.BeaconDB.StateSummary(ctx, bytesutil.ToBytes32(validatedCheckpoint.Root))
	if err != nil {
		return false, err
	}

	if ss.Slot > lastValidated.Slot {
		return true, nil
	}

	isCanonical, err := s.IsCanonical(ctx, root)
	if err != nil {
		return false, err
	}

	// historical non-canonical blocks here are returned as optimistic for safety.
	return !isCanonical, nil
}

// SetGenesisTime sets the genesis time of beacon chain.
func (s *Service) SetGenesisTime(t time.Time) {
	s.genesisTime = t
}

// ForkChoiceStore returns the fork choice store in the service
func (s *Service) ForkChoiceStore() forkchoice.ForkChoicer {
	return s.cfg.ForkChoiceStore
}
