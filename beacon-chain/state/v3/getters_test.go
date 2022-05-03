package v3

import (
	"runtime/debug"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	testtmpl "github.com/prysmaticlabs/prysm/beacon-chain/state/testing"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func TestBeaconState_SlotDataRace(t *testing.T) {
	testtmpl.VerifyBeaconStateSlotDataRace(t, func() (state.BeaconState, error) {
		return InitializeFromProto(&ethpb.BeaconStateBellatrix{Slot: 1})
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
	_ = st.GenesisValidatorsRoot()
	_ = st.GenesisValidatorsRoot()
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
	_, err = st.CurrentEpochParticipation()
	_ = err
	_, err = st.PreviousEpochParticipation()
	_ = err
	_ = st.JustificationBits()
	_ = st.PreviousJustifiedCheckpoint()
	_ = st.CurrentJustifiedCheckpoint()
	_ = st.FinalizedCheckpoint()
	_, err = st.CurrentEpochParticipation()
	_ = err
	_, err = st.PreviousEpochParticipation()
	_ = err
	_, err = st.InactivityScores()
	_ = err
	_, err = st.CurrentSyncCommittee()
	_ = err
	_, err = st.NextSyncCommittee()
	_ = err
}

func TestBeaconState_MatchCurrentJustifiedCheckpt(t *testing.T) {
	testtmpl.VerifyBeaconStateMatchCurrentJustifiedCheckpt(
		t,
		func(cp *ethpb.Checkpoint) (state.BeaconState, error) {
			return InitializeFromProto(&ethpb.BeaconStateBellatrix{CurrentJustifiedCheckpoint: cp})
		},
		func(i state.BeaconState) {
			s, ok := i.(*BeaconState)
			if !ok {
				panic("error in type assertion in test template")
			}
			s.state = nil
		},
	)
}

func TestBeaconState_MatchPreviousJustifiedCheckpt(t *testing.T) {
	testtmpl.VerifyBeaconStateMatchPreviousJustifiedCheckpt(
		t,
		func(cp *ethpb.Checkpoint) (state.BeaconState, error) {
			return InitializeFromProto(&ethpb.BeaconStateBellatrix{PreviousJustifiedCheckpoint: cp})
		},
		func(i state.BeaconState) {
			s, ok := i.(*BeaconState)
			if !ok {
				panic("error in type assertion in test template")
			}
			s.state = nil
		},
	)
}

func TestBeaconState_MarshalSSZ_NilState(t *testing.T) {
	testtmpl.VerifyBeaconStateMarshalSSZNilState(
		t,
		func() (state.BeaconState, error) {
			return InitializeFromProto(&ethpb.BeaconStateBellatrix{})
		},
		func(i state.BeaconState) {
			s, ok := i.(*BeaconState)
			if !ok {
				panic("error in type assertion in test template")
			}
			s.state = nil
		},
	)
}

func TestBeaconState_ValidatorByPubkey(t *testing.T) {
	testtmpl.VerifyBeaconStateValidatorByPubkey(t, func() (state.BeaconState, error) {
		return InitializeFromProto(&ethpb.BeaconStateBellatrix{})
	})
}
