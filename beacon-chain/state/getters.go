package state

import (
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Clone --
func (b *BeaconState) Clone() *pbp2p.BeaconState {
	return &pbp2p.BeaconState{
		GenesisTime:                 b.GenesisTime(),
		Slot:                        b.Slot(),
		Fork:                        b.Fork(),
		LatestBlockHeader:           b.LatestBlockHeader(),
		BlockRoots:                  b.BlockRoots(),
		StateRoots:                  b.StateRoots(),
		HistoricalRoots:             b.HistoricalRoots(),
		Eth1Data:                    b.Eth1Data(),
		Eth1DataVotes:               b.Eth1DataVotes(),
		Eth1DepositIndex:            b.Eth1DepositIndex(),
		Validators:                  b.Validators(),
		Balances:                    b.Balances(),
		RandaoMixes:                 b.RandaoMixes(),
		Slashings:                   b.Slashings(),
		PreviousEpochAttestations:   b.PreviousEpochAttestations(),
		CurrentEpochAttestations:    b.CurrentEpochAttestations(),
		JustificationBits:           b.JustificationBits(),
		PreviousJustifiedCheckpoint: b.PreviousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:  b.CurrentJustifiedCheckpoint(),
		FinalizedCheckpoint:         b.FinalizedCheckpoint(),
	}
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
	hdr := &ethpb.BeaconBlockHeader{
		Slot: b.state.LatestBlockHeader.Slot,
	}
	var parentRoot [32]byte
	var bodyRoot [32]byte
	var stateRoot [32]byte

	copy(parentRoot[:], b.state.LatestBlockHeader.ParentRoot)
	copy(bodyRoot[:], b.state.LatestBlockHeader.StateRoot)
	copy(stateRoot[:], b.state.LatestBlockHeader.BodyRoot)
	hdr.ParentRoot = parentRoot[:]
	hdr.BodyRoot = bodyRoot[:]
	hdr.StateRoot = stateRoot[:]
	return hdr
}

// BlockRoots --
func (b *BeaconState) BlockRoots() [][]byte {
	roots := make([][]byte, len(b.state.BlockRoots))
	for i, r := range b.state.BlockRoots {
		tmpRt := [32]byte{}
		copy(tmpRt[:], r)
		roots[i] = tmpRt[:]
	}
	return roots
}

// StateRoots --
func (b *BeaconState) StateRoots() [][]byte {
	roots := make([][]byte, len(b.state.StateRoots))
	for i, r := range b.state.StateRoots {
		tmpRt := [32]byte{}
		copy(tmpRt[:], r)
		roots[i] = tmpRt[:]
	}
	return roots
}

// HistoricalRoots --
func (b *BeaconState) HistoricalRoots() [][]byte {
	roots := make([][]byte, len(b.state.HistoricalRoots))
	for i, r := range b.state.HistoricalRoots {
		tmpRt := [32]byte{}
		copy(tmpRt[:], r)
		roots[i] = tmpRt[:]
	}
	return roots
}

// Eth1Data --
func (b *BeaconState) Eth1Data() *ethpb.Eth1Data {
	eth1data := &ethpb.Eth1Data{
		DepositCount: b.state.Eth1Data.DepositCount,
	}
	var depositRoot [32]byte
	var blockHash [32]byte

	copy(depositRoot[:], b.state.Eth1Data.DepositRoot)
	copy(blockHash[:], b.state.Eth1Data.BlockHash)

	eth1data.DepositRoot = depositRoot[:]
	eth1data.BlockHash = blockHash[:]

	return eth1data

}

// Eth1DataVotes --
func (b *BeaconState) Eth1DataVotes() []*ethpb.Eth1Data {
	res := make([]*ethpb.Eth1Data, len(b.state.Eth1DataVotes))
	for i := 0; i < len(res); i++ {
		res[i] = &ethpb.Eth1Data{
			DepositCount: b.state.Eth1Data.DepositCount,
		}
		var depositRoot [32]byte
		var blockHash [32]byte

		copy(depositRoot[:], b.state.Eth1DataVotes[i].DepositRoot)
		copy(blockHash[:], b.state.Eth1DataVotes[i].BlockHash)

		res[i].DepositRoot = depositRoot[:]
		res[i].BlockHash = blockHash[:]
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
	mixes := make([][]byte, len(b.state.RandaoMixes))
	for i, r := range b.state.RandaoMixes {
		tmpRt := [32]byte{}
		copy(tmpRt[:], r)
		mixes[i] = tmpRt[:]
	}
	return mixes
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

// PreviousJustifiedCheckpoint --
func (b *BeaconState) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: b.state.PreviousJustifiedCheckpoint.Epoch,
		Root:  b.state.PreviousJustifiedCheckpoint.Root,
	}
}

// CurrentJustifiedCheckpoint --
func (b *BeaconState) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: b.state.CurrentJustifiedCheckpoint.Epoch,
		Root:  b.state.CurrentJustifiedCheckpoint.Root,
	}
}

// FinalizedCheckpoint --
func (b *BeaconState) FinalizedCheckpoint() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{
		Epoch: b.state.FinalizedCheckpoint.Epoch,
		Root:  b.state.FinalizedCheckpoint.Root,
	}
}
