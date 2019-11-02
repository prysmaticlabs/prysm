package kafka

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func (e Exporter) DatabasePath() string {
	return e.db.DatabasePath()
}

func (e Exporter) ClearDB() error {
	return e.db.ClearDB()
}

func (e Exporter) Backup(ctx context.Context) error {
	return e.db.Backup(ctx)
}

func (e Exporter) AttestationsByDataRoot(ctx context.Context, attDataRoot [32]byte) ([]*eth.Attestation, error) {
	return e.db.AttestationsByDataRoot(ctx, attDataRoot)
}

func (e Exporter) Attestations(ctx context.Context, f *filters.QueryFilter) ([]*eth.Attestation, error) {
	return e.db.Attestations(ctx, f)
}

func (e Exporter) HasAttestation(ctx context.Context, attDataRoot [32]byte) bool {
	return e.db.HasAttestation(ctx, attDataRoot)
}

func (e Exporter) DeleteAttestation(ctx context.Context, attDataRoot [32]byte) error {
	return e.db.DeleteAttestation(ctx, attDataRoot)
}

func (e Exporter) DeleteAttestations(ctx context.Context, attDataRoots [][32]byte) error {
	return e.db.DeleteAttestations(ctx, attDataRoots)
}

func (e Exporter) Block(ctx context.Context, blockRoot [32]byte) (*eth.BeaconBlock, error) {
	return e.db.Block(ctx, blockRoot)
}

func (e Exporter) HeadBlock(ctx context.Context) (*eth.BeaconBlock, error) {
	return e.db.HeadBlock(ctx)
}

func (e Exporter) Blocks(ctx context.Context, f *filters.QueryFilter) ([]*eth.BeaconBlock, error) {
	return e.db.Blocks(ctx, f)
}

func (e Exporter) BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][32]byte, error) {
	return e.db.BlockRoots(ctx, f)
}

func (e Exporter) HasBlock(ctx context.Context, blockRoot [32]byte) bool {
	return e.db.HasBlock(ctx, blockRoot)
}

func (e Exporter) DeleteBlock(ctx context.Context, blockRoot [32]byte) error {
	return e.db.DeleteBlock(ctx, blockRoot)
}

func (e Exporter) DeleteBlocks(ctx context.Context, blockRoots [][32]byte) error {
	return e.db.DeleteBlocks(ctx, blockRoots)
}

func (e Exporter) ValidatorLatestVote(ctx context.Context, validatorIdx uint64) (*ethereum_beacon_p2p_v1.ValidatorLatestVote, error) {
	return e.db.ValidatorLatestVote(ctx, validatorIdx)
}

func (e Exporter) HasValidatorLatestVote(ctx context.Context, validatorIdx uint64) bool {
	return e.db.HasValidatorLatestVote(ctx, validatorIdx)
}

func (e Exporter) DeleteValidatorLatestVote(ctx context.Context, validatorIdx uint64) error {
	return e.db.DeleteValidatorLatestVote(ctx, validatorIdx)
}

func (e Exporter) ValidatorIndex(ctx context.Context, publicKey [48]byte) (uint64, bool, error) {
	return e.db.ValidatorIndex(ctx, publicKey)
}

func (e Exporter) HasValidatorIndex(ctx context.Context, publicKey [48]byte) bool {
	return e.db.HasValidatorIndex(ctx, publicKey)
}

func (e Exporter) DeleteValidatorIndex(ctx context.Context, publicKey [48]byte) error {
	return e.db.DeleteValidatorIndex(ctx, publicKey)
}

func (e Exporter) State(ctx context.Context, blockRoot [32]byte) (*ethereum_beacon_p2p_v1.BeaconState, error) {
	return e.db.State(ctx, blockRoot)
}

func (e Exporter) HeadState(ctx context.Context) (*ethereum_beacon_p2p_v1.BeaconState, error) {
	return e.db.HeadState(ctx)
}

func (e Exporter) GenesisState(ctx context.Context) (*ethereum_beacon_p2p_v1.BeaconState, error) {
	return e.db.GenesisState(ctx)
}

func (e Exporter) ProposerSlashing(ctx context.Context, slashingRoot [32]byte) (*eth.ProposerSlashing, error) {
	return e.db.ProposerSlashing(ctx, slashingRoot)
}

func (e Exporter) AttesterSlashing(ctx context.Context, slashingRoot [32]byte) (*eth.AttesterSlashing, error) {
	return e.db.AttesterSlashing(ctx, slashingRoot)
}

func (e Exporter) HasProposerSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	return e.db.HasProposerSlashing(ctx, slashingRoot)
}

func (e Exporter) HasAttesterSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	return e.db.HasAttesterSlashing(ctx, slashingRoot)
}

func (e Exporter) DeleteProposerSlashing(ctx context.Context, slashingRoot [32]byte) error {
	return e.db.DeleteProposerSlashing(ctx, slashingRoot)
}

func (e Exporter) DeleteAttesterSlashing(ctx context.Context, slashingRoot [32]byte) error {
	return e.db.DeleteAttesterSlashing(ctx, slashingRoot)
}

func (e Exporter) VoluntaryExit(ctx context.Context, exitRoot [32]byte) (*eth.VoluntaryExit, error) {
	return e.db.VoluntaryExit(ctx, exitRoot)
}

func (e Exporter) HasVoluntaryExit(ctx context.Context, exitRoot [32]byte) bool {
	return e.db.HasVoluntaryExit(ctx, exitRoot)
}

func (e Exporter) DeleteVoluntaryExit(ctx context.Context, exitRoot [32]byte) error {
	return e.db.DeleteVoluntaryExit(ctx, exitRoot)
}

func (e Exporter) JustifiedCheckpoint(ctx context.Context) (*eth.Checkpoint, error) {
	return e.db.JustifiedCheckpoint(ctx)
}

func (e Exporter) FinalizedCheckpoint(ctx context.Context) (*eth.Checkpoint, error) {
	return e.db.FinalizedCheckpoint(ctx)
}

func (e Exporter) ArchivedActiveValidatorChanges(ctx context.Context, epoch uint64) (*eth.ArchivedActiveSetChanges, error) {
	return e.db.ArchivedActiveValidatorChanges(ctx, epoch)
}

func (e Exporter) ArchivedCommitteeInfo(ctx context.Context, epoch uint64) (*eth.ArchivedCommitteeInfo, error) {
	return e.db.ArchivedCommitteeInfo(ctx, epoch)
}

func (e Exporter) ArchivedBalances(ctx context.Context, epoch uint64) ([]uint64, error) {
	return e.db.ArchivedBalances(ctx, epoch)
}

func (e Exporter) ArchivedValidatorParticipation(ctx context.Context, epoch uint64) (*eth.ValidatorParticipation, error) {
	return e.db.ArchivedValidatorParticipation(ctx, epoch)
}

func (e Exporter) DepositContractAddress(ctx context.Context) ([]byte, error) {
	return e.db.DepositContractAddress(ctx)
}

func (e Exporter) SaveHeadBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	return e.db.SaveHeadBlockRoot(ctx, blockRoot)
}

func (e Exporter) SaveGenesisBlockRoot(ctx context.Context, blockRoot [32]byte) error {
	return e.db.SaveGenesisBlockRoot(ctx, blockRoot)
}

func (e Exporter) SaveValidatorLatestVote(ctx context.Context, validatorIdx uint64, vote *ethereum_beacon_p2p_v1.ValidatorLatestVote) error {
	return e.db.SaveValidatorLatestVote(ctx, validatorIdx, vote)
}

func (e Exporter) SaveValidatorLatestVotes(ctx context.Context, validatorIndices []uint64, votes []*ethereum_beacon_p2p_v1.ValidatorLatestVote) error {
	return e.db.SaveValidatorLatestVotes(ctx, validatorIndices, votes)
}

func (e Exporter) SaveValidatorIndex(ctx context.Context, publicKey [48]byte, validatorIdx uint64) error {
	return e.db.SaveValidatorIndex(ctx, publicKey, validatorIdx)
}

func (e Exporter) SaveState(ctx context.Context, state *ethereum_beacon_p2p_v1.BeaconState, blockRoot [32]byte) error {
	return e.db.SaveState(ctx, state, blockRoot)
}

func (e Exporter) SaveProposerSlashing(ctx context.Context, slashing *eth.ProposerSlashing) error {
	return e.db.SaveProposerSlashing(ctx, slashing)
}

func (e Exporter) SaveAttesterSlashing(ctx context.Context, slashing *eth.AttesterSlashing) error {
	return e.db.SaveAttesterSlashing(ctx, slashing)
}

func (e Exporter) SaveVoluntaryExit(ctx context.Context, exit *eth.VoluntaryExit) error {
	return e.db.SaveVoluntaryExit(ctx, exit)
}

func (e Exporter) SaveJustifiedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error {
	return e.db.SaveJustifiedCheckpoint(ctx, checkpoint)
}

func (e Exporter) SaveFinalizedCheckpoint(ctx context.Context, checkpoint *eth.Checkpoint) error {
	return e.db.SaveFinalizedCheckpoint(ctx, checkpoint)
}

func (e Exporter) SaveArchivedActiveValidatorChanges(ctx context.Context, epoch uint64, changes *eth.ArchivedActiveSetChanges) error {
	return e.db.SaveArchivedActiveValidatorChanges(ctx, epoch, changes)
}

func (e Exporter) SaveArchivedCommitteeInfo(ctx context.Context, epoch uint64, info *eth.ArchivedCommitteeInfo) error {
	return e.db.SaveArchivedCommitteeInfo(ctx, epoch, info)
}

func (e Exporter) SaveArchivedBalances(ctx context.Context, epoch uint64, balances []uint64) error {
	return e.db.SaveArchivedBalances(ctx, epoch, balances)
}

func (e Exporter) SaveArchivedValidatorParticipation(ctx context.Context, epoch uint64, part *eth.ValidatorParticipation) error {
	return e.db.SaveArchivedValidatorParticipation(ctx, epoch, part)
}

func (e Exporter) SaveDepositContractAddress(ctx context.Context, addr common.Address) error {
	return e.db.SaveDepositContractAddress(ctx, addr)
}

func (e Exporter) DeleteState(ctx context.Context, blockRoot [32]byte) error {
	return e.db.DeleteState(ctx, blockRoot)
}

func (e Exporter) DeleteStates(ctx context.Context, blockRoots [][32]byte) error {
	return e.db.DeleteStates(ctx, blockRoots)
}

func (e Exporter) IsFinalizedBlock(ctx context.Context, blockRoot [32]byte) bool {
	return e.db.IsFinalizedBlock(ctx, blockRoot)
}
