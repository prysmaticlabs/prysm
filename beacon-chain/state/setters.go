package state

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func (b *BeaconState) SetGenesisTime(val uint64) {
	b.state.GenesisTime = val
}

func (b *BeaconState) SetSlot(val uint64) {
	b.state.Slot = val
}

func (b *BeaconState) SetFork(val *pbp2p.Fork) {
	b.state.Fork = val
}

func (b *BeaconState) SetLatestBlockHeader(val *ethpb.BeaconBlockHeader) {
	b.state.LatestBlockHeader = val
}

func (b *BeaconState) SetBlockRoots(val [][]byte) {
	b.state.BlockRoots = val
}

func (b *BeaconState) SetStateRoots(val [][]byte) {
	b.state.StateRoots = val
}

func (b *BeaconState) SetHistoricalRoots(val [][]byte) {
	b.state.HistoricalRoots = val
}

func (b *BeaconState) SetEth1Data(val *ethpb.Eth1Data) {
	b.state.Eth1Data = val
}

func (b *BeaconState) SetEth1DataVotes(val []*ethpb.Eth1Data) {
	b.state.Eth1DataVotes = val
}

func (b *BeaconState) SetEth1DepositIndex(val uint64) {
	b.state.Eth1DepositIndex = val
}

func (b *BeaconState) SetValidators(val []*ethpb.Validator) {
	b.state.Validators = val
}

func (b *BeaconState) SetBalances(val []uint64) {
	b.state.Balances = val
}

func (b *BeaconState) SetRandaoMixes(val [][]byte) {
	b.state.RandaoMixes = val
}

func (b *BeaconState) SetSlashings(val []uint64) {
	b.state.Slashings = val
}

func (b *BeaconState) SetPreviousEpochAttestations(val []*pbp2p.PendingAttestation) {
	b.state.PreviousEpochAttestations = val
}

func (b *BeaconState) SetCurrentEpochAttestations(val []*pbp2p.PendingAttestation) {
	b.state.CurrentEpochAttestations = val
}

func (b *BeaconState) SetJustificationBits(val bitfield.Bitvector4) {
	b.state.JustificationBits = val
}

func (b *BeaconState) SetPreviousJustifiedCheckpoint(val *ethpb.Checkpoint) {
	b.state.PreviousJustifiedCheckpoint = val
}

func (b *BeaconState) SetCurrentJustifiedCheckpoint(val *ethpb.Checkpoint) {
	b.state.CurrentJustifiedCheckpoint = val
}

func (b *BeaconState) SetFinalizedCheckpoint(val *ethpb.Checkpoint) {
	b.state.FinalizedCheckpoint = val
}
