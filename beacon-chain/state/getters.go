package state

import (
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// Clone the beacon state into a protobuf for usage.
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

// GenesisTime of the beacon state as a uint64.
func (b *BeaconState) GenesisTime() uint64 {
	return b.state.GenesisTime
}

// Slot of the current beacon chain state.
func (b *BeaconState) Slot() uint64 {
	return b.state.Slot
}

// Fork version of the beacon chain.
func (b *BeaconState) Fork() *pbp2p.Fork {
	if b.state.Fork == nil {
		return nil
	}
	prevVersion := make([]byte, len(b.state.Fork.PreviousVersion))
	copy(prevVersion, b.state.Fork.PreviousVersion)
	currVersion := make([]byte, len(b.state.Fork.PreviousVersion))
	copy(currVersion, b.state.Fork.PreviousVersion)
	return &pbp2p.Fork{
		PreviousVersion: prevVersion,
		CurrentVersion:  currVersion,
		Epoch:           b.state.Fork.Epoch,
	}
}

// LatestBlockHeader stored within the beacon state.
func (b *BeaconState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	if b.state.LatestBlockHeader == nil {
		return nil
	}
	hdr := &ethpb.BeaconBlockHeader{
		Slot: b.state.LatestBlockHeader.Slot,
	}
	var parentRoot [32]byte
	var bodyRoot [32]byte
	var stateRoot [32]byte

	copy(parentRoot[:], b.state.LatestBlockHeader.ParentRoot)
	copy(bodyRoot[:], b.state.LatestBlockHeader.BodyRoot)
	copy(stateRoot[:], b.state.LatestBlockHeader.StateRoot)
	hdr.ParentRoot = parentRoot[:]
	hdr.BodyRoot = bodyRoot[:]
	hdr.StateRoot = stateRoot[:]
	return hdr
}

// BlockRoots kept track of in the beacon state.
func (b *BeaconState) BlockRoots() [][]byte {
	if b.state.BlockRoots == nil {
		return nil
	}
	roots := make([][]byte, len(b.state.BlockRoots))
	for i, r := range b.state.BlockRoots {
		tmpRt := [32]byte{}
		copy(tmpRt[:], r)
		roots[i] = tmpRt[:]
	}
	return roots
}

// StateRoots kept track of in the beacon state.
func (b *BeaconState) StateRoots() [][]byte {
	if b.state.StateRoots == nil {
		return nil
	}
	roots := make([][]byte, len(b.state.StateRoots))
	for i, r := range b.state.StateRoots {
		tmpRt := [32]byte{}
		copy(tmpRt[:], r)
		roots[i] = tmpRt[:]
	}
	return roots
}

// HistoricalRoots based on epochs stored in the beacon state.
func (b *BeaconState) HistoricalRoots() [][]byte {
	if b.state.HistoricalRoots == nil {
		return nil
	}
	roots := make([][]byte, len(b.state.HistoricalRoots))
	for i, r := range b.state.HistoricalRoots {
		tmpRt := [32]byte{}
		copy(tmpRt[:], r)
		roots[i] = tmpRt[:]
	}
	return roots
}

// Eth1Data corresponding to the proof-of-work chain information stored in the beacon state.
func (b *BeaconState) Eth1Data() *ethpb.Eth1Data {
	if b.state.Eth1Data == nil {
		return nil
	}
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

// Eth1DataVotes corresponds to votes from eth2 on the canonical proof-of-work chain
// data retrieved from eth1.
func (b *BeaconState) Eth1DataVotes() []*ethpb.Eth1Data {
	if b.state.Eth1DataVotes == nil {
		return nil
	}
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

// Eth1DepositIndex corresponds to the index of the deposit made to the
// validator deposit contract at the time of this state's eth1 data.
func (b *BeaconState) Eth1DepositIndex() uint64 {
	return b.state.Eth1DepositIndex
}

// Validators participating in consensus on the beacon chain.
func (b *BeaconState) Validators() []*ethpb.Validator {
	if b.state.Validators == nil {
		return nil
	}
	res := make([]*ethpb.Validator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		val := b.state.Validators[i]
		var pubKey [48]byte
		copy(pubKey[:], val.PublicKey)
		var withdrawalCreds [32]byte
		copy(withdrawalCreds[:], val.WithdrawalCredentials)
		res[i] = &ethpb.Validator{
			PublicKey:                  pubKey[:],
			WithdrawalCredentials:      withdrawalCreds[:],
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

// Balances of validators participating in consensus on the beacon chain.
func (b *BeaconState) Balances() []uint64 {
	if b.state.Balances == nil {
		return nil
	}
	res := make([]uint64, len(b.state.Balances))
	copy(res, b.state.Balances)
	return res
}

// RandaoMixes of block proposers on the beacon chain.
func (b *BeaconState) RandaoMixes() [][]byte {
	if b.state.RandaoMixes == nil {
		return nil
	}
	mixes := make([][]byte, len(b.state.RandaoMixes))
	for i, r := range b.state.RandaoMixes {
		tmpRt := [32]byte{}
		copy(tmpRt[:], r)
		mixes[i] = tmpRt[:]
	}
	return mixes
}

// Slashings of validators on the beacon chain.
func (b *BeaconState) Slashings() []uint64 {
	if b.state.Slashings == nil {
		return nil
	}
	res := make([]uint64, len(b.state.Slashings))
	copy(res, b.state.Slashings)
	return res
}

// PreviousEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) PreviousEpochAttestations() []*pbp2p.PendingAttestation {
	if b.state.PreviousEpochAttestations == nil {
		return nil
	}
	res := make([]*pbp2p.PendingAttestation, len(b.state.PreviousEpochAttestations))
	for i := 0; i < len(res); i++ {
		res[i] = clonePendingAttestation(b.state.PreviousEpochAttestations[i])
	}
	return res
}

// CurrentEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) CurrentEpochAttestations() []*pbp2p.PendingAttestation {
	if b.state.CurrentEpochAttestations == nil {
		return nil
	}
	res := make([]*pbp2p.PendingAttestation, len(b.state.CurrentEpochAttestations))
	for i := 0; i < len(res); i++ {
		res[i] = clonePendingAttestation(b.state.CurrentEpochAttestations[i])
	}
	return res
}

// JustificationBits marking which epochs have been justified in the beacon chain.
func (b *BeaconState) JustificationBits() bitfield.Bitvector4 {
	if b.state.JustificationBits == nil {
		return nil
	}
	res := make([]byte, len(b.state.JustificationBits.Bytes()))
	copy(res, b.state.JustificationBits.Bytes())
	return res
}

// PreviousJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) PreviousJustifiedCheckpoint() *ethpb.Checkpoint {
	if b.state.PreviousJustifiedCheckpoint == nil {
		return nil
	}
	cp := &ethpb.Checkpoint{
		Epoch: b.state.PreviousJustifiedCheckpoint.Epoch,
	}
	var root [32]byte
	copy(root[:], b.state.PreviousJustifiedCheckpoint.Root)
	cp.Root = root[:]
	return cp
}

// CurrentJustifiedCheckpoint denoting an epoch and block root.
func (b *BeaconState) CurrentJustifiedCheckpoint() *ethpb.Checkpoint {
	if b.state.CurrentJustifiedCheckpoint == nil {
		return nil
	}
	cp := &ethpb.Checkpoint{
		Epoch: b.state.CurrentJustifiedCheckpoint.Epoch,
	}
	var root [32]byte
	copy(root[:], b.state.CurrentJustifiedCheckpoint.Root)
	cp.Root = root[:]
	return cp
}

// FinalizedCheckpoint denoting an epoch and block root.
func (b *BeaconState) FinalizedCheckpoint() *ethpb.Checkpoint {
	if b.state.FinalizedCheckpoint == nil {
		return nil
	}
	cp := &ethpb.Checkpoint{
		Epoch: b.state.FinalizedCheckpoint.Epoch,
	}
	var root [32]byte
	copy(root[:], b.state.FinalizedCheckpoint.Root)
	cp.Root = root[:]
	return cp
}

func clonePendingAttestation(att *pbp2p.PendingAttestation) *pbp2p.PendingAttestation {
	var aggBits bitfield.Bitlist
	copy(aggBits, att.AggregationBits)

	var attData *ethpb.AttestationData
	if att.Data != nil {
		var beaconRoot [32]byte
		copy(beaconRoot[:], att.Data.BeaconBlockRoot)

		var sourceRoot [32]byte
		copy(sourceRoot[:], att.Data.Source.Root)

		var targetRoot [32]byte
		copy(targetRoot[:], att.Data.Target.Root)
		attData = &ethpb.AttestationData{
			Slot:            att.Data.Slot,
			CommitteeIndex:  att.Data.CommitteeIndex,
			BeaconBlockRoot: beaconRoot[:],
			Source: &ethpb.Checkpoint{
				Epoch: att.Data.Source.Epoch,
				Root:  sourceRoot[:],
			},
			Target: &ethpb.Checkpoint{
				Epoch: att.Data.Target.Epoch,
				Root:  targetRoot[:],
			},
		}
	}
	return &pbp2p.PendingAttestation{
		AggregationBits: aggBits,
		Data:            attData,
		InclusionDelay:  att.InclusionDelay,
		ProposerIndex:   att.ProposerIndex,
	}
}
