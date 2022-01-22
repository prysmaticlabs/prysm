package v1

import (
	"runtime/debug"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	testtmpl "github.com/prysmaticlabs/prysm/beacon-chain/state/testing"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestBeaconState_SlotDataRace(t *testing.T) {
	testtmpl.VerifyBeaconState_SlotDataRace(t, func() (state.BeaconState, error) {
		return InitializeFromProto(&ethpb.BeaconState{Slot: 1})
	})
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
	_, _ = st.ValidatorIndexByPubkey([fieldparams.BLSPubkeyLength]byte{})
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
	_, err = st.PreviousEpochAttestations()
	require.NoError(t, err)
	_, err = st.CurrentEpochAttestations()
	require.NoError(t, err)
	_ = st.JustificationBits()
	_ = st.PreviousJustifiedCheckpoint()
	_ = st.CurrentJustifiedCheckpoint()
	_ = st.FinalizedCheckpoint()
}

func TestBeaconState_MatchCurrentJustifiedCheckpt(t *testing.T) {
	testtmpl.VerifyBeaconState_MatchCurrentJustifiedCheckpt(
		t,
		func(cp *ethpb.Checkpoint) (state.BeaconState, error) {
			return InitializeFromProto(&ethpb.BeaconState{CurrentJustifiedCheckpoint: cp})
		},
		func(i state.BeaconState) {
			s := i.(*BeaconState)
			s.state = nil
		},
	)
}

func TestBeaconState_MatchPreviousJustifiedCheckpt(t *testing.T) {
	testtmpl.VerifyBeaconState_MatchPreviousJustifiedCheckpt(
		t,
		func(cp *ethpb.Checkpoint) (state.BeaconState, error) {
			return InitializeFromProto(&ethpb.BeaconState{PreviousJustifiedCheckpoint: cp})
		},
		func(i state.BeaconState) {
			s := i.(*BeaconState)
			s.state = nil
		},
	)
}

func TestBeaconState_MarshalSSZ_NilState(t *testing.T) {
	testtmpl.VerifyBeaconState_MarshalSSZ_NilState(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProto(&ethpb.BeaconState{})
		},
		func(i state.BeaconState) {
			s := i.(*BeaconState)
			s.state = nil
		},
	)
}

func TestBeaconState_ValidatorByPubkey(t *testing.T) {
	testtmpl.VerifyBeaconState_ValidatorByPubkey(t, func() (state.BeaconState, error) {
		return InitializeFromProto(&ethpb.BeaconState{})
	})
}
