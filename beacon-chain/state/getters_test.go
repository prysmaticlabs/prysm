package state

import (
	"runtime/debug"
	"sync"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestBeaconState_SlotDataRace(t *testing.T) {
	headState, err := InitializeFromProto(&pb.BeaconState{Slot: 1})
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		if err := headState.SetSlot(uint64(0)); err != nil {
			t.Fatal(err)
		}
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
	_ = st.GenesisUnixTime()
	_ = st.GenesisValidatorRoot()
	_ = st.Slot()
	_ = st.Fork()
	_ = st.LatestBlockHeader()
	_ = st.ParentRoot()
	_ = st.BlockRoots()
	_, err := st.BlockRootAtIndex(0)
	_ = st.StateRoots()
	_ = st.HistoricalRoots()
	_ = st.Eth1Data()
	_ = st.Eth1DataVotes()
	_ = st.Eth1DepositIndex()
	_ = st.ValidatorsReadOnly()
	_, err = st.ValidatorAtIndex(0)
	_, err = st.ValidatorAtIndexReadOnly(0)
	_, _ = st.ValidatorIndexByPubkey([48]byte{})
	_ = st.validatorIndexMap()
	_ = st.PubkeyAtIndex(0)
	_ = st.NumValidators()
	_ = st.Balances()
	_, err = st.BalanceAtIndex(0)
	_ = st.BalancesLength()
	_ = st.RandaoMixes()
	_, err = st.RandaoMixAtIndex(0)
	_ = st.RandaoMixesLength()
	_ = st.Slashings()
	_ = st.PreviousEpochAttestations()
	_ = st.CurrentEpochAttestations()
	_ = st.JustificationBits()
	_ = st.PreviousJustifiedCheckpoint()
	_ = st.CurrentJustifiedCheckpoint()
	_ = st.FinalizedCheckpoint()
	_ = err
}

func TestReadOnlyValidator_NoPanic(t *testing.T) {
	v := &ReadOnlyValidator{}
	if v.Slashed() == true {
		t.Error("Expected not slashed")
	}
	if v.CopyValidator() != nil {
		t.Error("Expected nil result")
	}
}

func TestReadOnlyValidator_ActivationEligibilityEpochNoPanic(t *testing.T) {
	v := &ReadOnlyValidator{}
	if v.ActivationEligibilityEpoch() != 0 {
		t.Error("Expected 0 and not panic")
	}
}
