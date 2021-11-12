package v2

import (
	"github.com/pkg/errors"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ToProtoUnsafe returns the pointer value of the underlying
// beacon state proto object, bypassing immutability. Use with care.
func (b *BeaconState) ToProtoUnsafe() interface{} {
	if b == nil {
		return nil
	}

	bRoots := make([][]byte, len(b.blockRoots))
	for i, r := range b.blockRoots {
		bRoots[i] = r[:]
	}
	sRoots := make([][]byte, len(b.stateRoots))
	for i, r := range b.stateRoots {
		sRoots[i] = r[:]
	}
	hRoots := make([][]byte, len(b.historicalRoots))
	for i, r := range b.historicalRoots {
		hRoots[i] = r[:]
	}
	mixes := make([][]byte, len(b.randaoMixes))
	for i, m := range b.randaoMixes {
		mixes[i] = m[:]
	}
	return &ethpb.BeaconStateAltair{
		GenesisTime:                 b.genesisTime,
		GenesisValidatorsRoot:       b.genesisValidatorsRoot[:],
		Slot:                        b.slot,
		Fork:                        b.fork,
		LatestBlockHeader:           b.latestBlockHeader,
		BlockRoots:                  bRoots,
		StateRoots:                  sRoots,
		HistoricalRoots:             hRoots,
		Eth1Data:                    b.eth1Data,
		Eth1DataVotes:               b.eth1DataVotes,
		Eth1DepositIndex:            b.eth1DepositIndex,
		Validators:                  b.validators,
		Balances:                    b.balances,
		RandaoMixes:                 mixes,
		Slashings:                   b.slashings,
		PreviousEpochParticipation:  b.previousEpochParticipation,
		CurrentEpochParticipation:   b.currentEpochParticipation,
		JustificationBits:           b.justificationBits,
		PreviousJustifiedCheckpoint: b.previousJustifiedCheckpoint,
		CurrentJustifiedCheckpoint:  b.currentJustifiedCheckpoint,
		FinalizedCheckpoint:         b.finalizedCheckpoint,
		InactivityScores:            b.inactivityScores,
		CurrentSyncCommittee:        b.currentSyncCommittee,
		NextSyncCommittee:           b.nextSyncCommittee,
	}
}

// ToProto the beacon state into a protobuf for usage.
func (b *BeaconState) ToProto() interface{} {
	if b == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	gvr := b.genesisValidatorRootInternal()
	bRoots := make([][]byte, len(b.blockRootsInternal()))
	for i, r := range b.blockRootsInternal() {
		bRoots[i] = r[:]
	}
	sRoots := make([][]byte, len(b.stateRootsInternal()))
	for i, r := range b.stateRootsInternal() {
		sRoots[i] = r[:]
	}
	hRoots := make([][]byte, len(b.historicalRootsInternal()))
	for i, r := range b.historicalRootsInternal() {
		hRoots[i] = r[:]
	}
	mixes := make([][]byte, len(b.randaoMixesInternal()))
	for i, m := range b.randaoMixesInternal() {
		mixes[i] = m[:]
	}
	return &ethpb.BeaconStateAltair{
		GenesisTime:                 b.genesisTimeInternal(),
		GenesisValidatorsRoot:       gvr[:],
		Slot:                        b.slotInternal(),
		Fork:                        b.forkInternal(),
		LatestBlockHeader:           b.latestBlockHeaderInternal(),
		BlockRoots:                  bRoots,
		StateRoots:                  sRoots,
		HistoricalRoots:             hRoots,
		Eth1Data:                    b.eth1DataInternal(),
		Eth1DataVotes:               b.eth1DataVotesInternal(),
		Eth1DepositIndex:            b.eth1DepositIndexInternal(),
		Validators:                  b.validatorsInternal(),
		Balances:                    b.balancesInternal(),
		RandaoMixes:                 mixes,
		Slashings:                   b.slashingsInternal(),
		PreviousEpochParticipation:  b.previousEpochParticipationInternal(),
		CurrentEpochParticipation:   b.currentEpochParticipationInternal(),
		JustificationBits:           b.justificationBitsInternal(),
		PreviousJustifiedCheckpoint: b.previousJustifiedCheckpointInternal(),
		CurrentJustifiedCheckpoint:  b.currentJustifiedCheckpointInternal(),
		FinalizedCheckpoint:         b.finalizedCheckpointInternal(),
		InactivityScores:            b.inactivityScoresInternal(),
		CurrentSyncCommittee:        b.currentSyncCommitteeInternal(),
		NextSyncCommittee:           b.nextSyncCommitteeInternal(),
	}
}

// hasInnerState detects if the internal reference to the state data structure
// is populated correctly. Returns false if nil.
func (b *BeaconState) hasInnerState() bool {
	// TODO: Remove this function entirely
	return true
}

// StateRoots kept track of in the beacon state.
func (b *BeaconState) StateRoots() *[8192][32]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.stateRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	roots := [8192][32]byte(*b.stateRootsInternal())
	return &roots
}

// stateRootsInternal kept track of in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) stateRootsInternal() *customtypes.StateRoots {
	if !b.hasInnerState() {
		return nil
	}
	return b.stateRoots
}

// StateRootAtIndex retrieves a specific state root based on an
// input index value.
func (b *BeaconState) StateRootAtIndex(idx uint64) ([32]byte, error) {
	if !b.hasInnerState() {
		return [32]byte{}, ErrNilInnerState
	}
	if b.stateRoots == nil {
		return [32]byte{}, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.stateRootAtIndex(idx)
}

// stateRootAtIndex retrieves a specific state root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) stateRootAtIndex(idx uint64) ([32]byte, error) {
	if !b.hasInnerState() {
		return [32]byte{}, ErrNilInnerState
	}
	sRoots := make([][]byte, len(b.stateRoots))
	for i := range sRoots {
		sRoots[i] = b.stateRoots[i][:]
	}
	root, err := bytesutil.SafeCopyRootAtIndex(sRoots, idx)
	if err != nil {
		return [32]byte{}, err
	}
	return bytesutil.ToBytes32(root), nil
}

// ProtobufBeaconState transforms an input into beacon state hard fork 1 in the form of protobuf.
// Error is returned if the input is not type protobuf beacon state.
func ProtobufBeaconState(s interface{}) (*ethpb.BeaconStateAltair, error) {
	pbState, ok := s.(*ethpb.BeaconStateAltair)
	if !ok {
		return nil, errors.New("input is not type pb.BeaconStateAltair")
	}
	return pbState, nil
}
