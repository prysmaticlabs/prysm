package state

import (
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Clone --
func (b *BeaconState) Clone() *pbp2p.BeaconState {
	return proto.Clone(b.state).(*pbp2p.BeaconState)
}

// GenesisTime --
func (b *BeaconState) GenesisTime() uint64 {
	return b.state.GenesisTime
}

// Slot --
func (b *BeaconState) Slot() uint64 {
	return b.state.Slot
}

// Fork --
func (b *BeaconState) Fork() *pbp2p.Fork {
	return &pbp2p.Fork{
		PreviousVersion: b.state.Fork.PreviousVersion,
		CurrentVersion:  b.state.Fork.CurrentVersion,
		Epoch:           b.state.Fork.Epoch,
	}
}

// LatestBlockHeader --
func (b *BeaconState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	return &ethpb.BeaconBlockHeader{
		Slot:       b.state.LatestBlockHeader.Slot,
		ParentRoot: b.state.LatestBlockHeader.ParentRoot,
		StateRoot:  b.state.LatestBlockHeader.StateRoot,
		BodyRoot:   b.state.LatestBlockHeader.BodyRoot,
	}
}

// BlockRoots --
func (b *BeaconState) BlockRoots() [][]byte {
	res := make([][]byte, len(b.state.BlockRoots))
	copy(res, b.state.BlockRoots)
	return res
}

// StateRoots --
func (b *BeaconState) StateRoots() [][]byte {
	res := make([][]byte, len(b.state.StateRoots))
	copy(res, b.state.StateRoots)
	return res
}

// HistoricalRoots --
func (b *BeaconState) HistoricalRoots() [][]byte {
	res := make([][]byte, len(b.state.HistoricalRoots))
	copy(res, b.state.HistoricalRoots)
	return res
}

// Eth1Data --
func (b *BeaconState) Eth1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{
		DepositRoot:  b.state.Eth1Data.DepositRoot,
		DepositCount: b.state.Eth1Data.DepositCount,
		BlockHash:    b.state.Eth1Data.BlockHash,
	}
}

// Eth1DataVotes --
func (b *BeaconState) Eth1DataVotes() []*ethpb.Eth1Data {
	res := make([]*ethpb.Eth1Data, len(b.state.Eth1DataVotes))
	for i := 0; i < len(res); i++ {
		res[i] = &ethpb.Eth1Data{
			DepositRoot:  b.state.Eth1DataVotes[i].DepositRoot,
			DepositCount: b.state.Eth1DataVotes[i].DepositCount,
			BlockHash:    b.state.Eth1DataVotes[i].BlockHash,
		}
	}
	return res
}

// Eth1DepositIndex --
func (b *BeaconState) Eth1DepositIndex() uint64 {
	return b.state.Eth1DepositIndex
}

// Validators --
func (b *BeaconState) Validators() []*ethpb.Validator {
	res := make([]*ethpb.Validator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		val := b.state.Validators[i]
		res[i] = &ethpb.Validator{
			PublicKey:                  val.PublicKey,
			WithdrawalCredentials:      val.WithdrawalCredentials,
			EffectiveBalance:           val.EffectiveBalance,
			Slashed:                    val.Slashed,
			ActivationEligibilityEpoch: val.ActivationEligibilityEpoch,
			ActivationEpoch:            val.ActivationEpoch,
			ExitEpoch:                  val.ExitEpoch,
			WithdrawableEpoch:          val.WithdrawableEpoch,
		}
	}
	return res
}

// Balances --
func (b *BeaconState) Balances() []uint64 {
	res := make([]uint64, len(b.state.Balances))
	copy(res, b.state.Balances)
	return res
}

// RandaoMixes --
func (b *BeaconState) RandaoMixes() [][]byte {
	res := make([][]byte, len(b.state.RandaoMixes))
	copy(res, b.state.RandaoMixes)
	return res
}

// Slashings --
func (b *BeaconState) Slashings() []uint64 {
	res := make([]uint64, len(b.state.Slashings))
	copy(res, b.state.Slashings)
	return res
}

// PreviousEpochAttestations --
func (b *BeaconState) PreviousEpochAttestations() []*pbp2p.PendingAttestation {
	res := make([]*pbp2p.PendingAttestation, len(b.state.PreviousEpochAttestations))
	for i := 0; i < len(res); i++ {
		res[i] = proto.Clone(b.state.PreviousEpochAttestations[i]).(*pbp2p.PendingAttestation)
	}
	return res
}

// CurrentEpochAttestations --
func (b *BeaconState) CurrentEpochAttestations() []*pbp2p.PendingAttestation {
	res := make([]*pbp2p.PendingAttestation, len(b.state.CurrentEpochAttestations))
	for i := 0; i < len(res); i++ {
		res[i] = proto.Clone(b.state.CurrentEpochAttestations[i]).(*pbp2p.PendingAttestation)
	}
	return res
}

// JustificationBits --
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
