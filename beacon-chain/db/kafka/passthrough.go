package kafka

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/common"
	types "github.com/prysmaticlabs/eth2-types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/proto/beacon/db"
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
func (e Exporter) Backup(ctx context.Context, outputDir string) error {
	return e.db.Backup(ctx, outputDir)
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
func (e Exporter) Blocks(ctx context.Context, f *filters.QueryFilter) ([]*eth.SignedBeaconBlock, [][32]byte, error) {
	return e.db.Blocks(ctx, f)
}

// BlockRoots -- passthrough.
func (e Exporter) BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][32]byte, error) {
	return e.db.BlockRoots(ctx, f)
}

// BlocksBySlot -- passthrough.
func (e Exporter) BlocksBySlot(ctx context.Context, slot types.Slot) (bool, []*eth.SignedBeaconBlock, error) {
	return e.db.BlocksBySlot(ctx, slot)
}

// BlockRootsBySlot -- passthrough.
func (e Exporter) BlockRootsBySlot(ctx context.Context, slot types.Slot) (bool, [][32]byte, error) {
	return e.db.BlockRootsBySlot(ctx, slot)
}

// HasBlock -- passthrough.
func (e Exporter) HasBlock(ctx context.Context, blockRoot [32]byte) bool {
	return e.db.HasBlock(ctx, blockRoot)
}

// State -- passthrough.
func (e Exporter) State(ctx context.Context, blockRoot [32]byte) (iface.BeaconState, error) {
	return e.db.State(ctx, blockRoot)
}

// StateSummary -- passthrough.
func (e Exporter) StateSummary(ctx context.Context, blockRoot [32]byte) (*pb.StateSummary, error) {
	return e.db.StateSummary(ctx, blockRoot)
}

// GenesisState -- passthrough.
func (e Exporter) GenesisState(ctx context.Context) (iface.BeaconState, error) {
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

// DepositContractAddress -- passthrough.
func (e Exporter) DepositContractAddress(ctx context.Context) ([]byte, error) {
	return e.db.DepositContractAddress(ctx)
}

// SaveHeadBlockRoot -- passthrough.
func (e Exporter) SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	return e.db.SaveHeadBlockRoot(ctx, blockRoot)
}

// GenesisBlock -- passthrough.
func (e Exporter) GenesisBlock(ctx context.Context) (*eth.SignedBeaconBlock, error) {
	return e.db.GenesisBlock(ctx)
}

// SaveGenesisBlockRoot -- passthrough.
func (e Exporter) SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	return e.db.SaveGenesisBlockRoot(ctx, blockRoot)
}

// SaveState -- passthrough.
func (e Exporter) SaveState(ctx context.Context, st iface.ReadOnlyBeaconState, blockRoot [32]byte) error {
	return e.db.SaveState(ctx, st, blockRoot)
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
func (e Exporter) SaveStates(ctx context.Context, states []iface.ReadOnlyBeaconState, blockRoots [][32]byte) error {
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

// FinalizedChildBlock -- passthrough.
func (e Exporter) FinalizedChildBlock(ctx context.Context, blockRoot [32]byte) (*eth.SignedBeaconBlock, error) {
	return e.db.FinalizedChildBlock(ctx, blockRoot)
}

// PowchainData -- passthrough
func (e Exporter) PowchainData(ctx context.Context) (*db.ETH1ChainData, error) {
	return e.db.PowchainData(ctx)
}

// SavePowchainData -- passthrough
func (e Exporter) SavePowchainData(ctx context.Context, data *db.ETH1ChainData) error {
	return e.db.SavePowchainData(ctx, data)
}

// ArchivedPointRoot -- passthrough
func (e Exporter) ArchivedPointRoot(ctx context.Context, index types.Slot) [32]byte {
	return e.db.ArchivedPointRoot(ctx, index)
}

// HasArchivedPoint -- passthrough
func (e Exporter) HasArchivedPoint(ctx context.Context, index types.Slot) bool {
	return e.db.HasArchivedPoint(ctx, index)
}

// LastArchivedRoot -- passthrough
func (e Exporter) LastArchivedRoot(ctx context.Context) [32]byte {
	return e.db.LastArchivedRoot(ctx)
}

// HighestSlotBlocksBelow -- passthrough
func (e Exporter) HighestSlotBlocksBelow(ctx context.Context, slot types.Slot) ([]*eth.SignedBeaconBlock, error) {
	return e.db.HighestSlotBlocksBelow(ctx, slot)
}

// HighestSlotStatesBelow -- passthrough
func (e Exporter) HighestSlotStatesBelow(ctx context.Context, slot types.Slot) ([]iface.ReadOnlyBeaconState, error) {
	return e.db.HighestSlotStatesBelow(ctx, slot)
}

// LastArchivedSlot -- passthrough
func (e Exporter) LastArchivedSlot(ctx context.Context) (types.Slot, error) {
	return e.db.LastArchivedSlot(ctx)
}

// RunMigrations -- passthrough
func (e Exporter) RunMigrations(ctx context.Context) error {
	return e.db.RunMigrations(ctx)
}

// CleanUpDirtyStates -- passthrough
func (e Exporter) CleanUpDirtyStates(ctx context.Context, slotsPerArchivedPoint types.Slot) error {
	return e.db.RunMigrations(ctx)
}

// LoadGenesisFromFile -- passthrough
func (e Exporter) LoadGenesis(ctx context.Context, r io.Reader) error {
	return e.db.LoadGenesis(ctx, r)
}

// SaveGenesisData -- passthrough
func (e Exporter) SaveGenesisData(ctx context.Context, state iface.BeaconState) error {
	return e.db.SaveGenesisData(ctx, state)
}

// EnsureEmbeddedGenesis -- passthrough.
func (e Exporter) EnsureEmbeddedGenesis(ctx context.Context) error {
	return e.db.EnsureEmbeddedGenesis(ctx)
}
