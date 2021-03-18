package stateV0

import (
	"runtime/debug"
	"sync"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBeaconState_SlotDataRace(t *testing.T) {
	headState, err := InitializeFromProto(&pb.BeaconState{Slot: 1})
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		require.NoError(t, headState.SetSlot(0))
		wg.Done()
	}()
	go func() {
		headState.Slot()
		wg.Done()
	}()

	wg.Wait()
}

func TestNilState_NoPanic(t *testing.T) {
	var st *BeaconState
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Method panicked when it was not supposed to: %v\n%v\n", r, string(debug.Stack()))
		}
	}()
	// retrieve elements from nil state
	_ = st.GenesisTime()
	_ = st.GenesisValidatorRoot()
	_ = st.GenesisValidatorRoot()
	_ = st.Slot()
	_ = st.Fork()
	_ = st.LatestBlockHeader()
	_ = st.BlockRoots()
	_, err := st.BlockRootAtIndex(0)
	_ = err
	_ = st.StateRoots()
	_ = st.HistoricalRoots()
	_ = st.Eth1Data()
	_ = st.Eth1DataVotes()
	_ = st.Eth1DepositIndex()
	_, err = st.ValidatorAtIndex(0)
	_ = err
	_, err = st.ValidatorAtIndexReadOnly(0)
	_ = err
	_, _ = st.ValidatorIndexByPubkey([48]byte{})
	_ = st.validatorIndexMap()
	_ = st.PubkeyAtIndex(0)
	_ = st.NumValidators()
	_ = st.Balances()
	_, err = st.BalanceAtIndex(0)
	_ = err
	_ = st.BalancesLength()
	_ = st.RandaoMixes()
	_, err = st.RandaoMixAtIndex(0)
	_ = err
	_ = st.RandaoMixesLength()
	_ = st.Slashings()
	_ = st.PreviousEpochAttestations()
	_ = st.CurrentEpochAttestations()
	_ = st.JustificationBits()
	_ = st.PreviousJustifiedCheckpoint()
	_ = st.CurrentJustifiedCheckpoint()
	_ = st.FinalizedCheckpoint()
}

func TestReadOnlyValidator_NoPanic(t *testing.T) {
	v := &ReadOnlyValidator{}
	assert.Equal(t, false, v.Slashed(), "Expected not slashed")
}

func TestReadOnlyValidator_ActivationEligibilityEpochNoPanic(t *testing.T) {
	v := &ReadOnlyValidator{}
	assert.Equal(t, types.Epoch(0), v.ActivationEligibilityEpoch(), "Expected 0 and not panic")
}

func TestBeaconState_MatchCurrentJustifiedCheckpt(t *testing.T) {
	c1 := &eth.Checkpoint{Epoch: 1}
	c2 := &eth.Checkpoint{Epoch: 2}
	beaconState, err := InitializeFromProto(&pb.BeaconState{CurrentJustifiedCheckpoint: c1})
	require.NoError(t, err)
	require.Equal(t, true, beaconState.MatchCurrentJustifiedCheckpoint(c1))
	require.Equal(t, false, beaconState.MatchCurrentJustifiedCheckpoint(c2))
	require.Equal(t, false, beaconState.MatchPreviousJustifiedCheckpoint(c1))
	require.Equal(t, false, beaconState.MatchPreviousJustifiedCheckpoint(c2))
	beaconState.state = nil
	require.Equal(t, false, beaconState.MatchCurrentJustifiedCheckpoint(c1))
}

func TestBeaconState_MatchPreviousJustifiedCheckpt(t *testing.T) {
	c1 := &eth.Checkpoint{Epoch: 1}
	c2 := &eth.Checkpoint{Epoch: 2}
	beaconState, err := InitializeFromProto(&pb.BeaconState{PreviousJustifiedCheckpoint: c1})
	require.NoError(t, err)
	require.NoError(t, err)
	require.Equal(t, false, beaconState.MatchCurrentJustifiedCheckpoint(c1))
	require.Equal(t, false, beaconState.MatchCurrentJustifiedCheckpoint(c2))
	require.Equal(t, true, beaconState.MatchPreviousJustifiedCheckpoint(c1))
	require.Equal(t, false, beaconState.MatchPreviousJustifiedCheckpoint(c2))
	beaconState.state = nil
	require.Equal(t, false, beaconState.MatchPreviousJustifiedCheckpoint(c1))
}

func TestBeaconState_MarshalSSZ_NilState(t *testing.T) {
	s, err := InitializeFromProto(&pb.BeaconState{})
	require.NoError(t, err)
	s.state = nil
	_, err = s.MarshalSSZ()
	require.ErrorContains(t, "nil beacon state", err)
}
