package state

import (
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func (b *BeaconState) Clone() pbp2p.BeaconState {
	return b.state
}

func (b *BeaconState) GenesisTime() uint64 {
	return b.state.GenesisTime
}

func (b *BeaconState) Slot() uint64 {
	return b.state.Slot
}

func (b *BeaconState) Fork() *pbp2p.Fork {
	return proto.Clone(b.state.Fork).(*pbp2p.Fork)
}

func (b *BeaconState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	return proto.Clone(b.state.LatestBlockHeader).(*ethpb.BeaconBlockHeader)
}

func (b *BeaconState) BlockRoots() [][]byte {
	res := make([][]byte, len(b.state.BlockRoots))
	copy(res, b.state.BlockRoots)
	return res
}

func (b *BeaconState) StateRoots() [][]byte {
	res := make([][]byte, len(b.state.StateRoots))
	copy(res, b.state.StateRoots)
	return res
}

func (b *BeaconState) HistoricalRoots() [][]byte {
	res := make([][]byte, len(b.state.HistoricalRoots))
	copy(res, b.state.HistoricalRoots)
	return res
}

func (b *BeaconState) Eth1Data() *ethpb.Eth1Data {
	return proto.Clone(b.state.Eth1Data).(*ethpb.Eth1Data)
}

func (b *BeaconState) Eth1DataVotes() []*ethpb.Eth1Data {
	// TODO: Clone this value.
	return b.state.Eth1DataVotes
}

func (b *BeaconState) Eth1DepositIndex() uint64 {
	return b.state.Eth1DepositIndex
}

func (b *BeaconState) Validators() []*ethpb.Validator {
	// TODO: Clone this value.
	return b.state.Validators
}

func (b *BeaconState) Balances() []uint64 {
	res := make([]uint64, len(b.state.Balances))
	copy(res, b.state.Balances)
	return res
}

func (b *BeaconState) RandaoMixes() [][]byte {
	res := make([][]byte, len(b.state.RandaoMixes))
	copy(res, b.state.RandaoMixes)
	return res
}

func (b *BeaconState) Slashings() []uint64 {
	res := make([]uint64, len(b.state.Slashings))
	copy(res, b.state.Slashings)
	return res
}

func (b *BeaconState) PreviousEpochAttestations() []*pbp2p.PendingAttestation {
	// TODO: Clone this value.
	return b.state.PreviousEpochAttestations
}

func (b *BeaconState) CurrentEpochAttestations() []*pbp2p.PendingAttestation {
	// TODO: Clone this value.
	return b.state.CurrentEpochAttestations
}

func (b *BeaconState) JustificationBits() bitfield.Bitvector4 {
	res := bitfield.Bitvector4{}
	copy(res, b.state.JustificationBits)
	return res
}

func (b *BeaconState) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: b.state.PreviousJustifiedCheckpoint.Epoch,
		Root:  b.state.PreviousJustifiedCheckpoint.Root,
	}
}

func (b *BeaconState) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: b.state.CurrentJustifiedCheckpoint.Epoch,
		Root:  b.state.CurrentJustifiedCheckpoint.Root,
	}
}

func (b *BeaconState) FinalizedCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: b.state.FinalizedCheckpoint.Epoch,
		Root:  b.state.FinalizedCheckpoint.Root,
	}
}
