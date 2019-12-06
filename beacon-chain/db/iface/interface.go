// Package iface exists to prevent circular dependencies when implementing the database interface.
package iface

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// Database defines the necessary methods for Prysm's eth2 backend which may
// be implemented by any key-value or relational database in practice.
type Database interface {
	io.Closer
	DatabasePath() string
	ClearDB() error
	// Backup and restore methods
	Backup(ctx context.Context) error
	// Attestation related methods.
	AttestationsByDataRoot(ctx context.Context, attDataRoot [32]byte) ([]*eth.Attestation, error)
	Attestations(ctx context.Context, f *filters.QueryFilter) ([]*eth.Attestation, error)
	HasAttestation(ctx context.Context, attDataRoot [32]byte) bool
	DeleteAttestation(ctx context.Context, attDataRoot [32]byte) error
	DeleteAttestations(ctx context.Context, attDataRoots [][32]byte) error
	SaveAttestation(ctx context.Context, att *eth.Attestation) error
	SaveAttestations(ctx context.Context, atts []*eth.Attestation) error
	// Block related methods.
	Block(ctx context.Context, blockRoot [32]byte) (*eth.BeaconBlock, error)
	HeadBlock(ctx context.Context) (*eth.BeaconBlock, error)
	Blocks(ctx context.Context, f *filters.QueryFilter) ([]*eth.BeaconBlock, error)
	BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][32]byte, error)
	HasBlock(ctx context.Context, blockRoot [32]byte) bool
	DeleteBlock(ctx context.Context, blockRoot [32]byte) error
	DeleteBlocks(ctx context.Context, blockRoots [][32]byte) error
	SaveBlock(ctx context.Context, block *eth.BeaconBlock) error
	SaveBlocks(ctx context.Context, blocks []*eth.BeaconBlock) error
	SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error
	SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error
	IsFinalizedBlock(ctx context.Context, blockRoot [32]byte) bool
	// Validator related methods.
	ValidatorIndex(ctx context.Context, publicKey [48]byte) (uint64, bool, error)
	HasValidatorIndex(ctx context.Context, publicKey [48]byte) bool
	DeleteValidatorIndex(ctx context.Context, publicKey [48]byte) error
	SaveValidatorIndex(ctx context.Context, publicKey [48]byte, validatorIdx uint64) error
	// State related methods.
	State(ctx context.Context, blockRoot [32]byte) (*ethereum_beacon_p2p_v1.BeaconState, error)
	HeadState(ctx context.Context) (*ethereum_beacon_p2p_v1.BeaconState, error)
	GenesisState(ctx context.Context) (*ethereum_beacon_p2p_v1.BeaconState, error)
	SaveState(ctx context.Context, state *ethereum_beacon_p2p_v1.BeaconState, blockRoot [32]byte) error
	DeleteState(ctx context.Context, blockRoot [32]byte) error
	DeleteStates(ctx context.Context, blockRoots [][32]byte) error
	// Slashing operations.
	ProposerSlashing(ctx context.Context, slashingRoot [32]byte) (*eth.ProposerSlashing, error)
	AttesterSlashing(ctx context.Context, slashingRoot [32]byte) (*eth.AttesterSlashing, error)
	SaveProposerSlashing(ctx context.Context, slashing *eth.ProposerSlashing) error
	SaveAttesterSlashing(ctx context.Context, slashing *eth.AttesterSlashing) error
	HasProposerSlashing(ctx context.Context, slashingRoot [32]byte) bool
	HasAttesterSlashing(ctx context.Context, slashingRoot [32]byte) bool
	DeleteProposerSlashing(ctx context.Context, slashingRoot [32]byte) error
	DeleteAttesterSlashing(ctx context.Context, slashingRoot [32]byte) error
	// Block operations.
	VoluntaryExit(ctx context.Context, exitRoot [32]byte) (*eth.VoluntaryExit, error)
	SaveVoluntaryExit(ctx context.Context, exit *eth.VoluntaryExit) error
	HasVoluntaryExit(ctx context.Context, exitRoot [32]byte) bool
	DeleteVoluntaryExit(ctx context.Context, exitRoot [32]byte) error
	// Checkpoint operations.
	JustifiedCheckpoint(ctx context.Context) (*eth.Checkpoint, error)
	FinalizedCheckpoint(ctx context.Context) (*eth.Checkpoint, error)
	SaveJustifiedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error
	SaveFinalizedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error
	// Archival data handlers for storing/retrieving historical beacon node information.
	ArchivedActiveValidatorChanges(ctx context.Context, epoch uint64) (*ethereum_beacon_p2p_v1.ArchivedActiveSetChanges, error)
	SaveArchivedActiveValidatorChanges(ctx context.Context, epoch uint64, changes *ethereum_beacon_p2p_v1.ArchivedActiveSetChanges) error
	ArchivedCommitteeInfo(ctx context.Context, epoch uint64) (*ethereum_beacon_p2p_v1.ArchivedCommitteeInfo, error)
	SaveArchivedCommitteeInfo(ctx context.Context, epoch uint64, info *ethereum_beacon_p2p_v1.ArchivedCommitteeInfo) error
	ArchivedBalances(ctx context.Context, epoch uint64) ([]uint64, error)
	SaveArchivedBalances(ctx context.Context, epoch uint64, balances []uint64) error
	ArchivedValidatorParticipation(ctx context.Context, epoch uint64) (*eth.ValidatorParticipation, error)
	SaveArchivedValidatorParticipation(ctx context.Context, epoch uint64, part *eth.ValidatorParticipation) error
	// Deposit contract related handlers.
	DepositContractAddress(ctx context.Context) ([]byte, error)
	SaveDepositContractAddress(ctx context.Context, addr common.Address) error
}
