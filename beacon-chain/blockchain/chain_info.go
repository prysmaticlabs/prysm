package blockchain

import (
	"bytes"
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

// ChainInfoFetcher defines a common interface for methods in blockchain service which
// directly retrieve chain info related data.
type ChainInfoFetcher interface {
	HeadFetcher
	FinalizationFetcher
	CanonicalFetcher
	ForkFetcher
	HeadDomainFetcher
}

// HeadUpdater defines a common interface for methods in blockchain service
// which allow to update the head info
type HeadUpdater interface {
	UpdateHead(context.Context) error
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
// directly retrieve head related data.
type HeadFetcher interface {
	HeadSlot() types.Slot
	HeadRoot(ctx context.Context) ([]byte, error)
	HeadBlock(ctx context.Context) (interfaces.SignedBeaconBlock, error)
	HeadState(ctx context.Context) (state.BeaconState, error)
	HeadValidatorsIndices(ctx context.Context, epoch types.Epoch) ([]types.ValidatorIndex, error)
	HeadGenesisValidatorsRoot() [32]byte
	HeadETH1Data() *ethpb.Eth1Data
	HeadPublicKeyToValidatorIndex(pubKey [fieldparams.BLSPubkeyLength]byte) (types.ValidatorIndex, bool)
	HeadValidatorIndexToPublicKey(ctx context.Context, index types.ValidatorIndex) ([fieldparams.BLSPubkeyLength]byte, error)
	ChainHeads() ([][32]byte, []types.Slot)
	HeadSyncCommitteeFetcher
	HeadDomainFetcher
}

// ForkFetcher retrieves the current fork information of the Ethereum beacon chain.
type ForkFetcher interface {
	ForkChoicer() forkchoice.ForkChoicer
	CurrentFork() *ethpb.Fork
	GenesisFetcher
	TimeFetcher
}

// CanonicalFetcher retrieves the current chain's canonical information.
type CanonicalFetcher interface {
	IsCanonical(ctx context.Context, blockRoot [32]byte) (bool, error)
}

// FinalizationFetcher defines a common interface for methods in blockchain service which
// directly retrieve finalization and justification related data.
type FinalizationFetcher interface {
	FinalizedCheckpt() *ethpb.Checkpoint
	CurrentJustifiedCheckpt() *ethpb.Checkpoint
	PreviousJustifiedCheckpt() *ethpb.Checkpoint
	VerifyFinalizedBlkDescendant(ctx context.Context, blockRoot [32]byte) error
	IsFinalized(ctx context.Context, blockRoot [32]byte) bool
}

// OptimisticModeFetcher retrieves information about optimistic status of the node.
type OptimisticModeFetcher interface {
	IsOptimistic(ctx context.Context) (bool, error)
	IsOptimisticForRoot(ctx context.Context, root [32]byte) (bool, error)
}

// FinalizedCheckpt returns the latest finalized checkpoint from chain store.
func (s *Service) FinalizedCheckpt() *ethpb.Checkpoint {
	cp := s.ForkChoicer().FinalizedCheckpoint()
	return &ethpb.Checkpoint{Epoch: cp.Epoch, Root: bytesutil.SafeCopyBytes(cp.Root[:])}
}

// PreviousJustifiedCheckpt returns the current justified checkpoint from chain store.
func (s *Service) PreviousJustifiedCheckpt() *ethpb.Checkpoint {
	cp := s.ForkChoicer().PreviousJustifiedCheckpoint()
	return &ethpb.Checkpoint{Epoch: cp.Epoch, Root: bytesutil.SafeCopyBytes(cp.Root[:])}
}

// CurrentJustifiedCheckpt returns the current justified checkpoint from chain store.
func (s *Service) CurrentJustifiedCheckpt() *ethpb.Checkpoint {
	cp := s.ForkChoicer().JustifiedCheckpoint()
	return &ethpb.Checkpoint{Epoch: cp.Epoch, Root: bytesutil.SafeCopyBytes(cp.Root[:])}
}

// BestJustifiedCheckpt returns the best justified checkpoint from store.
func (s *Service) BestJustifiedCheckpt() *ethpb.Checkpoint {
	cp := s.ForkChoicer().BestJustifiedCheckpoint()
	return &ethpb.Checkpoint{Epoch: cp.Epoch, Root: bytesutil.SafeCopyBytes(cp.Root[:])}
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

	if s.head != nil && s.head.root != params.BeaconConfig().ZeroHash {
		return bytesutil.SafeCopyBytes(s.head.root[:]), nil
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
func (s *Service) HeadBlock(ctx context.Context) (interfaces.SignedBeaconBlock, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	if s.hasHeadState() {
		return s.headBlock()
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

// GenesisValidatorsRoot returns the genesis validators
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
	// If the block has not been finalized, check fork choice store to see if the block is canonical
	if s.cfg.ForkChoiceStore.HasNode(blockRoot) {
		return s.cfg.ForkChoiceStore.IsCanonical(blockRoot), nil
	}

	// If the block has been finalized, the block will always be part of the canonical chain.
	return s.cfg.BeaconDB.IsFinalizedBlock(ctx, blockRoot), nil
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

// ForkChoicer returns the forkchoice interface.
func (s *Service) ForkChoicer() forkchoice.ForkChoicer {
	return s.cfg.ForkChoiceStore
}

// IsOptimistic returns true if the current head is optimistic.
func (s *Service) IsOptimistic(ctx context.Context) (bool, error) {
	if slots.ToEpoch(s.CurrentSlot()) < params.BeaconConfig().BellatrixForkEpoch {
		return false, nil
	}
	s.headLock.RLock()
	headRoot := s.head.root
	s.headLock.RUnlock()

	if s.cfg.ForkChoiceStore.AllTipsAreInvalid() {
		return true, nil
	}
	optimistic, err := s.cfg.ForkChoiceStore.IsOptimistic(headRoot)
	if err == nil {
		return optimistic, nil
	}
	if err != protoarray.ErrUnknownNodeRoot && err != doublylinkedtree.ErrNilNode {
		return true, err
	}
	// If fockchoice does not have the headroot, then the node is considered
	// optimistic
	return true, nil
}

// IsFinalized returns true if the input root is finalized.
// It first checks latest finalized root then checks finalized root index in DB.
func (s *Service) IsFinalized(ctx context.Context, root [32]byte) bool {
	if s.ForkChoicer().FinalizedCheckpoint().Root == root {
		return true
	}
	return s.cfg.BeaconDB.IsFinalizedBlock(ctx, root)
}

// IsOptimisticForRoot takes the root as argument instead of the current head
// and returns true if it is optimistic.
func (s *Service) IsOptimisticForRoot(ctx context.Context, root [32]byte) (bool, error) {
	optimistic, err := s.cfg.ForkChoiceStore.IsOptimistic(root)
	if err == nil {
		return optimistic, nil
	}
	if err != protoarray.ErrUnknownNodeRoot && err != doublylinkedtree.ErrNilNode {
		return false, err
	}
	// if the requested root is the headroot and the root is not found in
	// forkchoice, the node should respond that it is optimistic
	headRoot, err := s.HeadRoot(ctx)
	if err != nil {
		return true, err
	}
	if bytes.Equal(headRoot, root[:]) {
		return true, nil
	}

	ss, err := s.cfg.BeaconDB.StateSummary(ctx, root)
	if err != nil {
		return false, err
	}

	if ss == nil {
		return true, errInvalidNilSummary
	}
	validatedCheckpoint, err := s.cfg.BeaconDB.LastValidatedCheckpoint(ctx)
	if err != nil {
		return false, err
	}
	if slots.ToEpoch(ss.Slot) > validatedCheckpoint.Epoch {
		return true, nil
	}

	// Historical non-canonical blocks here are returned as optimistic for safety.
	isCanonical, err := s.IsCanonical(ctx, root)
	if err != nil {
		return false, err
	}

	if slots.ToEpoch(ss.Slot)+1 < validatedCheckpoint.Epoch {
		return !isCanonical, nil
	}

	// Checkpoint root could be zeros before the first finalized epoch. Use genesis root if the case.
	lastValidated, err := s.cfg.BeaconDB.StateSummary(ctx, s.ensureRootNotZeros(bytesutil.ToBytes32(validatedCheckpoint.Root)))
	if err != nil {
		return false, err
	}
	if lastValidated == nil {
		return false, errInvalidNilSummary
	}

	if ss.Slot > lastValidated.Slot {
		return true, nil
	}
	return !isCanonical, nil
}

// SetGenesisTime sets the genesis time of beacon chain.
func (s *Service) SetGenesisTime(t time.Time) {
	s.genesisTime = t
}

// ForkChoiceStore returns the fork choice store in the service.
func (s *Service) ForkChoiceStore() forkchoice.ForkChoicer {
	return s.cfg.ForkChoiceStore
}
