package db

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// Database defines the necessary methods for Prysm's eth2 backend which may
// be implemented by any key-value or relational database in practice.
type Database interface {
	io.Closer
	DatabasePath() string
	ClearDB() error
	// Attestation related methods.
	Attestation(ctx context.Context, attRoot [32]byte) (*ethpb.Attestation, error)
	Attestations(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.Attestation, error)
	HasAttestation(ctx context.Context, attRoot [32]byte) bool
	DeleteAttestation(ctx context.Context, attRoot [32]byte) error
	SaveAttestation(ctx context.Context, att *ethpb.Attestation) error
	SaveAttestations(ctx context.Context, atts []*ethpb.Attestation) error
	// Block related methods.
	Block(ctx context.Context, blockRoot [32]byte) (*ethpb.BeaconBlock, error)
	HeadBlock(ctx context.Context) (*ethpb.BeaconBlock, error)
	Blocks(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.BeaconBlock, error)
	BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][]byte, error)
	HasBlock(ctx context.Context, blockRoot [32]byte) bool
	DeleteBlock(ctx context.Context, blockRoot [32]byte) error
	SaveBlock(ctx context.Context, block *ethpb.BeaconBlock) error
	SaveBlocks(ctx context.Context, blocks []*ethpb.BeaconBlock) error
	SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error
	SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error
	// Validator related methods.
	ValidatorLatestVote(ctx context.Context, validatorIdx uint64) (*pb.ValidatorLatestVote, error)
	HasValidatorLatestVote(ctx context.Context, validatorIdx uint64) bool
	DeleteValidatorLatestVote(ctx context.Context, validatorIdx uint64) error
	SaveValidatorLatestVote(ctx context.Context, validatorIdx uint64, vote *pb.ValidatorLatestVote) error
	ValidatorIndex(ctx context.Context, publicKey [48]byte) (uint64, bool, error)
	HasValidatorIndex(ctx context.Context, publicKey [48]byte) bool
	DeleteValidatorIndex(ctx context.Context, publicKey [48]byte) error
	SaveValidatorIndex(ctx context.Context, publicKey [48]byte, validatorIdx uint64) error
	// State related methods.
	State(ctx context.Context, blockRoot [32]byte) (*pb.BeaconState, error)
	HeadState(ctx context.Context) (*pb.BeaconState, error)
	GenesisState(ctx context.Context) (*pb.BeaconState, error)
	SaveState(ctx context.Context, state *pb.BeaconState, blockRoot [32]byte) error
	// Slashing operations.
	ProposerSlashing(ctx context.Context, slashingRoot [32]byte) (*ethpb.ProposerSlashing, error)
	AttesterSlashing(ctx context.Context, slashingRoot [32]byte) (*ethpb.AttesterSlashing, error)
	SaveProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) error
	SaveAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) error
	HasProposerSlashing(ctx context.Context, slashingRoot [32]byte) bool
	HasAttesterSlashing(ctx context.Context, slashingRoot [32]byte) bool
	DeleteProposerSlashing(ctx context.Context, slashingRoot [32]byte) error
	DeleteAttesterSlashing(ctx context.Context, slashingRoot [32]byte) error
	// Block operations.
	VoluntaryExit(ctx context.Context, exitRoot [32]byte) (*ethpb.VoluntaryExit, error)
	SaveVoluntaryExit(ctx context.Context, exit *ethpb.VoluntaryExit) error
	HasVoluntaryExit(ctx context.Context, exitRoot [32]byte) bool
	DeleteVoluntaryExit(ctx context.Context, exitRoot [32]byte) error
	// Checkpoint operations.
	JustifiedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error)
	FinalizedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error)
	SaveJustifiedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error
	SaveFinalizedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error
	// Deposit contract related handlers.
	DepositContractAddress(ctx context.Context) ([]byte, error)
	SaveDepositContractAddress(ctx context.Context, addr common.Address) error
}

// NewDB initializes a new DB.
func NewDB(dirPath string) (Database, error) {
	return kv.NewKVStore(dirPath)
}
