package kafka

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/proto/beacon/db"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// DatabasePath -- passthrough.
func (e Exporter) DatabasePath() string {
	return e.db.DatabasePath()
}

// ClearDB -- passthrough.
func (e Exporter) ClearDB() error {
	return e.db.ClearDB()
}

// Backup -- passthrough.
func (e Exporter) Backup(ctx context.Context) error {
	return e.db.Backup(ctx)
}

// AttestationsByDataRoot -- passthrough.
func (e Exporter) AttestationsByDataRoot(ctx context.Context, attDataRoot [32]byte) ([]*eth.Attestation, error) {
	return e.db.AttestationsByDataRoot(ctx, attDataRoot)
}

// Attestations -- passthrough.
func (e Exporter) Attestations(ctx context.Context, f *filters.QueryFilter) ([]*eth.Attestation, error) {
	return e.db.Attestations(ctx, f)
}

// HasAttestation -- passthrough.
func (e Exporter) HasAttestation(ctx context.Context, attDataRoot [32]byte) bool {
	return e.db.HasAttestation(ctx, attDataRoot)
}

// DeleteAttestation -- passthrough.
func (e Exporter) DeleteAttestation(ctx context.Context, attDataRoot [32]byte) error {
	return e.db.DeleteAttestation(ctx, attDataRoot)
}

// DeleteAttestations -- passthrough.
func (e Exporter) DeleteAttestations(ctx context.Context, attDataRoots [][32]byte) error {
	return e.db.DeleteAttestations(ctx, attDataRoots)
}

// Block -- passthrough.
func (e Exporter) Block(ctx context.Context, blockRoot [32]byte) (*eth.SignedBeaconBlock, error) {
	return e.db.Block(ctx, blockRoot)
}

// HeadBlock -- passthrough.
func (e Exporter) HeadBlock(ctx context.Context) (*eth.SignedBeaconBlock, error) {
	return e.db.HeadBlock(ctx)
}

// Blocks -- passthrough.
func (e Exporter) Blocks(ctx context.Context, f *filters.QueryFilter) ([]*eth.SignedBeaconBlock, error) {
	return e.db.Blocks(ctx, f)
}

// BlockRoots -- passthrough.
func (e Exporter) BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][32]byte, error) {
	return e.db.BlockRoots(ctx, f)
}

// HasBlock -- passthrough.
func (e Exporter) HasBlock(ctx context.Context, blockRoot [32]byte) bool {
	return e.db.HasBlock(ctx, blockRoot)
}

// State -- passthrough.
func (e Exporter) State(ctx context.Context, blockRoot [32]byte) (*state.BeaconState, error) {
	return e.db.State(ctx, blockRoot)
}

// StateSummary -- passthrough.
func (e Exporter) StateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	return e.db.StateSummary(ctx, blockRoot)
}

// HeadState -- passthrough.
func (e Exporter) HeadState(ctx context.Context) (*state.BeaconState, error) {
	return e.db.HeadState(ctx)
}

// GenesisState -- passthrough.
func (e Exporter) GenesisState(ctx context.Context) (*state.BeaconState, error) {
	return e.db.GenesisState(ctx)
}

// ProposerSlashing -- passthrough.
func (e Exporter) ProposerSlashing(ctx context.Context, slashingRoot [32]byte) (*eth.ProposerSlashing, error) {
	return e.db.ProposerSlashing(ctx, slashingRoot)
}

// AttesterSlashing -- passthrough.
func (e Exporter) AttesterSlashing(ctx context.Context, slashingRoot [32]byte) (*eth.AttesterSlashing, error) {
	return e.db.AttesterSlashing(ctx, slashingRoot)
}

// HasProposerSlashing -- passthrough.
func (e Exporter) HasProposerSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	return e.db.HasProposerSlashing(ctx, slashingRoot)
}

// HasAttesterSlashing -- passthrough.
func (e Exporter) HasAttesterSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	return e.db.HasAttesterSlashing(ctx, slashingRoot)
}

// VoluntaryExit -- passthrough.
func (e Exporter) VoluntaryExit(ctx context.Context, exitRoot [32]byte) (*eth.VoluntaryExit, error) {
	return e.db.VoluntaryExit(ctx, exitRoot)
}

// HasVoluntaryExit -- passthrough.
func (e Exporter) HasVoluntaryExit(ctx context.Context, exitRoot [32]byte) bool {
	return e.db.HasVoluntaryExit(ctx, exitRoot)
}

// JustifiedCheckpoint -- passthrough.
func (e Exporter) JustifiedCheckpoint(ctx context.Context) (*eth.Checkpoint, error) {
	return e.db.JustifiedCheckpoint(ctx)
}

// FinalizedCheckpoint -- passthrough.
func (e Exporter) FinalizedCheckpoint(ctx context.Context) (*eth.Checkpoint, error) {
	return e.db.FinalizedCheckpoint(ctx)
}

// ArchivedActiveValidatorChanges -- passthrough.
func (e Exporter) ArchivedActiveValidatorChanges(ctx context.Context, epoch uint64) (*ethereum_beacon_p2p_v1.ArchivedActiveSetChanges, error) {
	return e.db.ArchivedActiveValidatorChanges(ctx, epoch)
}

// ArchivedCommitteeInfo -- passthrough.
func (e Exporter) ArchivedCommitteeInfo(ctx context.Context, epoch uint64) (*ethereum_beacon_p2p_v1.ArchivedCommitteeInfo, error) {
	return e.db.ArchivedCommitteeInfo(ctx, epoch)
}

// ArchivedBalances -- passthrough.
func (e Exporter) ArchivedBalances(ctx context.Context, epoch uint64) ([]uint64, error) {
	return e.db.ArchivedBalances(ctx, epoch)
}

// ArchivedValidatorParticipation -- passthrough.
func (e Exporter) ArchivedValidatorParticipation(ctx context.Context, epoch uint64) (*eth.ValidatorParticipation, error) {
	return e.db.ArchivedValidatorParticipation(ctx, epoch)
}

// DepositContractAddress -- passthrough.
func (e Exporter) DepositContractAddress(ctx context.Context) ([]byte, error) {
	return e.db.DepositContractAddress(ctx)
}

// SaveHeadBlockRoot -- passthrough.
func (e Exporter) SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	return e.db.SaveHeadBlockRoot(ctx, blockRoot)
}

// GenesisBlock -- passthrough.
func (e Exporter) GenesisBlock(ctx context.Context) (*ethpb.SignedBeaconBlock, error) {
	return e.db.GenesisBlock(ctx)
}

// SaveGenesisBlockRoot -- passthrough.
func (e Exporter) SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	return e.db.SaveGenesisBlockRoot(ctx, blockRoot)
}

// SaveState -- passthrough.
func (e Exporter) SaveState(ctx context.Context, state *state.BeaconState, blockRoot [32]byte) error {
	return e.db.SaveState(ctx, state, blockRoot)
}

// SaveStateSummary -- passthrough.
func (e Exporter) SaveStateSummary(ctx context.Context, summary *pb.StateSummary) error {
	return e.db.SaveStateSummary(ctx, summary)
}

// SaveStateSummaries -- passthrough.
func (e Exporter) SaveStateSummaries(ctx context.Context, summaries []*pb.StateSummary) error {
	return e.db.SaveStateSummaries(ctx, summaries)
}

// SaveStates -- passthrough.
func (e Exporter) SaveStates(ctx context.Context, states []*state.BeaconState, blockRoots [][32]byte) error {
	return e.db.SaveStates(ctx, states, blockRoots)
}

// SaveProposerSlashing -- passthrough.
func (e Exporter) SaveProposerSlashing(ctx context.Context, slashing *eth.ProposerSlashing) error {
	return e.db.SaveProposerSlashing(ctx, slashing)
}

// SaveAttesterSlashing -- passthrough.
func (e Exporter) SaveAttesterSlashing(ctx context.Context, slashing *eth.AttesterSlashing) error {
	return e.db.SaveAttesterSlashing(ctx, slashing)
}

// SaveVoluntaryExit -- passthrough.
func (e Exporter) SaveVoluntaryExit(ctx context.Context, exit *eth.VoluntaryExit) error {
	return e.db.SaveVoluntaryExit(ctx, exit)
}

// SaveJustifiedCheckpoint -- passthrough.
func (e Exporter) SaveJustifiedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error {
	return e.db.SaveJustifiedCheckpoint(ctx, checkpoint)
}

// SaveFinalizedCheckpoint -- passthrough.
func (e Exporter) SaveFinalizedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error {
	return e.db.SaveFinalizedCheckpoint(ctx, checkpoint)
}

// SaveArchivedActiveValidatorChanges -- passthrough.
func (e Exporter) SaveArchivedActiveValidatorChanges(ctx context.Context, epoch uint64, changes *ethereum_beacon_p2p_v1.ArchivedActiveSetChanges) error {
	return e.db.SaveArchivedActiveValidatorChanges(ctx, epoch, changes)
}

// SaveArchivedCommitteeInfo -- passthrough.
func (e Exporter) SaveArchivedCommitteeInfo(ctx context.Context, epoch uint64, info *ethereum_beacon_p2p_v1.ArchivedCommitteeInfo) error {
	return e.db.SaveArchivedCommitteeInfo(ctx, epoch, info)
}

// SaveArchivedBalances -- passthrough.
func (e Exporter) SaveArchivedBalances(ctx context.Context, epoch uint64, balances []uint64) error {
	return e.db.SaveArchivedBalances(ctx, epoch, balances)
}

// SaveArchivedValidatorParticipation -- passthrough.
func (e Exporter) SaveArchivedValidatorParticipation(ctx context.Context, epoch uint64, part *eth.ValidatorParticipation) error {
	return e.db.SaveArchivedValidatorParticipation(ctx, epoch, part)
}

// SaveDepositContractAddress -- passthrough.
func (e Exporter) SaveDepositContractAddress(ctx context.Context, addr common.Address) error {
	return e.db.SaveDepositContractAddress(ctx, addr)
}

// DeleteState -- passthrough.
func (e Exporter) DeleteState(ctx context.Context, blockRoot [32]byte) error {
	return e.db.DeleteState(ctx, blockRoot)
}

// DeleteStates -- passthrough.
func (e Exporter) DeleteStates(ctx context.Context, blockRoots [][32]byte) error {
	return e.db.DeleteStates(ctx, blockRoots)
}

// HasState -- passthrough.
func (e Exporter) HasState(ctx context.Context, blockRoot [32]byte) bool {
	return e.db.HasState(ctx, blockRoot)
}

// HasStateSummary -- passthrough.
func (e Exporter) HasStateSummary(ctx context.Context, blockRoot [32]byte) bool {
	return e.db.HasStateSummary(ctx, blockRoot)
}

// IsFinalizedBlock -- passthrough.
func (e Exporter) IsFinalizedBlock(ctx context.Context, blockRoot [32]byte) bool {
	return e.db.IsFinalizedBlock(ctx, blockRoot)
}

// PowchainData -- passthrough
func (e Exporter) PowchainData(ctx context.Context) (*db.ETH1ChainData, error) {
	return e.db.PowchainData(ctx)
}

// SavePowchainData -- passthrough
func (e Exporter) SavePowchainData(ctx context.Context, data *db.ETH1ChainData) error {
	return e.db.SavePowchainData(ctx, data)
}

// SaveArchivedPointRoot -- passthrough
func (e Exporter) SaveArchivedPointRoot(ctx context.Context, blockRoot [32]byte, slot uint64) error {
	return e.db.SaveArchivedPointRoot(ctx, blockRoot, slot)
}

// ArchivedPointRoot -- passthrough
func (e Exporter) ArchivedPointRoot(ctx context.Context, index uint64) [32]byte {
	return e.db.ArchivedPointRoot(ctx, index)
}

// HasArchivedPoint -- passthrough
func (e Exporter) HasArchivedPoint(ctx context.Context, index uint64) bool {
	return e.db.HasArchivedPoint(ctx, index)
}

// LastArchivedRoot -- passthrough
func (e Exporter) LastArchivedRoot(ctx context.Context) [32]byte {
	return e.db.LastArchivedRoot(ctx)
}

// HighestSlotBlocks -- passthrough
func (e Exporter) HighestSlotBlocks(ctx context.Context) ([]*ethpb.SignedBeaconBlock, error) {
	return e.db.HighestSlotBlocks(ctx)
}

// HighestSlotBlocksBelow -- passthrough
func (e Exporter) HighestSlotBlocksBelow(ctx context.Context, slot uint64) ([]*ethpb.SignedBeaconBlock, error) {
	return e.db.HighestSlotBlocksBelow(ctx, slot)
}

// HighestSlotStatesBelow -- passthrough
func (e Exporter) HighestSlotStatesBelow(ctx context.Context, slot uint64) ([]*state.BeaconState, error) {
	return e.db.HighestSlotStatesBelow(ctx, slot)
}

// LastArchivedSlot -- passthrough
func (e Exporter) LastArchivedSlot(ctx context.Context) (uint64, error) {
	return e.db.LastArchivedSlot(ctx)
}

// HistoricalStatesDeleted -- passthrough
func (e Exporter) HistoricalStatesDeleted(ctx context.Context) error {
	return e.db.HistoricalStatesDeleted(ctx)
}
