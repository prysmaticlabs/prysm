// Package iface defines the actual database interface used
// by a Prysm beacon node, also containing useful, scoped interfaces such as
// a ReadOnlyDatabase.
package iface

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/common"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/monitoring/backup"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

// ReadOnlyDatabase defines a struct which only has read access to database methods.
type ReadOnlyDatabase interface {
	// Block related methods.
	Block(ctx context.Context, blockRoot [32]byte) (block.SignedBeaconBlock, error)
	Blocks(ctx context.Context, f *filters.QueryFilter) ([]block.SignedBeaconBlock, [][32]byte, error)
	BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][32]byte, error)
	BlocksBySlot(ctx context.Context, slot types.Slot) (bool, []block.SignedBeaconBlock, error)
	BlockRootsBySlot(ctx context.Context, slot types.Slot) (bool, [][32]byte, error)
	HasBlock(ctx context.Context, blockRoot [32]byte) bool
	GenesisBlock(ctx context.Context) (block.SignedBeaconBlock, error)
	IsFinalizedBlock(ctx context.Context, blockRoot [32]byte) bool
	FinalizedChildBlock(ctx context.Context, blockRoot [32]byte) (block.SignedBeaconBlock, error)
	HighestSlotBlocksBelow(ctx context.Context, slot types.Slot) ([]block.SignedBeaconBlock, error)
	// State related methods.
	State(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error)
	GenesisState(ctx context.Context) (state.BeaconState, error)
	HasState(ctx context.Context, blockRoot [32]byte) bool
	StateSummary(ctx context.Context, blockRoot [32]byte) (*ethpb.StateSummary, error)
	HasStateSummary(ctx context.Context, blockRoot [32]byte) bool
	HighestSlotStatesBelow(ctx context.Context, slot types.Slot) ([]state.ReadOnlyBeaconState, error)
	// Checkpoint operations.
	JustifiedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error)
	FinalizedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error)
	ArchivedPointRoot(ctx context.Context, slot types.Slot) [32]byte
	HasArchivedPoint(ctx context.Context, slot types.Slot) bool
	LastArchivedRoot(ctx context.Context) [32]byte
	LastArchivedSlot(ctx context.Context) (types.Slot, error)
	// Deposit contract related handlers.
	DepositContractAddress(ctx context.Context) ([]byte, error)
	// Powchain operations.
	PowchainData(ctx context.Context) (*ethpb.ETH1ChainData, error)

	// origin checkpoint sync support
	OriginBlockRoot(ctx context.Context) ([32]byte, error)
}

// NoHeadAccessDatabase defines a struct without access to chain head data.
type NoHeadAccessDatabase interface {
	ReadOnlyDatabase

	// Block related methods.
	SaveBlock(ctx context.Context, block block.SignedBeaconBlock) error
	SaveBlocks(ctx context.Context, blocks []block.SignedBeaconBlock) error
	SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error
	// State related methods.
	SaveState(ctx context.Context, state state.ReadOnlyBeaconState, blockRoot [32]byte) error
	SaveStates(ctx context.Context, states []state.ReadOnlyBeaconState, blockRoots [][32]byte) error
	DeleteState(ctx context.Context, blockRoot [32]byte) error
	DeleteStates(ctx context.Context, blockRoots [][32]byte) error
	SaveStateSummary(ctx context.Context, summary *ethpb.StateSummary) error
	SaveStateSummaries(ctx context.Context, summaries []*ethpb.StateSummary) error
	// Checkpoint operations.
	SaveJustifiedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error
	SaveFinalizedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error
	// Deposit contract related handlers.
	SaveDepositContractAddress(ctx context.Context, addr common.Address) error
	// Powchain operations.
	SavePowchainData(ctx context.Context, data *ethpb.ETH1ChainData) error
	// Run any required database migrations.
	RunMigrations(ctx context.Context) error

	CleanUpDirtyStates(ctx context.Context, slotsPerArchivedPoint types.Slot) error
}

// HeadAccessDatabase defines a struct with access to reading chain head data.
type HeadAccessDatabase interface {
	NoHeadAccessDatabase

	// Block related methods.
	HeadBlock(ctx context.Context) (block.SignedBeaconBlock, error)
	SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error

	// Genesis operations.
	LoadGenesis(ctx context.Context, r io.Reader) error
	SaveGenesisData(ctx context.Context, state state.BeaconState) error
	EnsureEmbeddedGenesis(ctx context.Context) error

	// initialization method needed for origin checkpoint sync
	SaveOrigin(ctx context.Context, state io.Reader, block io.Reader) error
}

// SlasherDatabase interface for persisting data related to detecting slashable offenses on Ethereum.
type SlasherDatabase interface {
	io.Closer
	SaveLastEpochsWrittenForValidators(
		ctx context.Context, epochByValidator map[types.ValidatorIndex]types.Epoch,
	) error
	SaveAttestationRecordsForValidators(
		ctx context.Context,
		attestations []*slashertypes.IndexedAttestationWrapper,
	) error
	SaveSlasherChunks(
		ctx context.Context, kind slashertypes.ChunkKind, chunkKeys [][]byte, chunks [][]uint16,
	) error
	SaveBlockProposals(
		ctx context.Context, proposal []*slashertypes.SignedBlockHeaderWrapper,
	) error
	LastEpochWrittenForValidators(
		ctx context.Context, validatorIndices []types.ValidatorIndex,
	) ([]*slashertypes.AttestedEpochForValidator, error)
	AttestationRecordForValidator(
		ctx context.Context, validatorIdx types.ValidatorIndex, targetEpoch types.Epoch,
	) (*slashertypes.IndexedAttestationWrapper, error)
	BlockProposalForValidator(
		ctx context.Context, validatorIdx types.ValidatorIndex, slot types.Slot,
	) (*slashertypes.SignedBlockHeaderWrapper, error)
	CheckAttesterDoubleVotes(
		ctx context.Context, attestations []*slashertypes.IndexedAttestationWrapper,
	) ([]*slashertypes.AttesterDoubleVote, error)
	LoadSlasherChunks(
		ctx context.Context, kind slashertypes.ChunkKind, diskKeys [][]byte,
	) ([][]uint16, []bool, error)
	CheckDoubleBlockProposals(
		ctx context.Context, proposals []*slashertypes.SignedBlockHeaderWrapper,
	) ([]*ethpb.ProposerSlashing, error)
	PruneAttestationsAtEpoch(
		ctx context.Context, maxEpoch types.Epoch,
	) (numPruned uint, err error)
	PruneProposalsAtEpoch(
		ctx context.Context, maxEpoch types.Epoch,
	) (numPruned uint, err error)
	HighestAttestations(
		ctx context.Context,
		indices []types.ValidatorIndex,
	) ([]*ethpb.HighestAttestation, error)
	DatabasePath() string
	ClearDB() error
}

// Database interface with full access.
type Database interface {
	io.Closer
	backup.BackupExporter
	HeadAccessDatabase

	DatabasePath() string
	ClearDB() error
}
