package state_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestInitializeFromProto(t *testing.T) {
	testState, _ := testutil.DeterministicGenesisState(t, 64)

	type test struct {
		name  string
		state *pbp2p.BeaconState
		error string
	}
	initTests := []test{
		{
			name:  "nil state",
			state: nil,
			error: "received nil state",
		},
		{
			name: "nil validators",
			state: &pbp2p.BeaconState{
				Slot:       4,
				Validators: nil,
			},
			error: "",
		},
		{
			name:  "empty state",
			state: &pbp2p.BeaconState{},
			error: "",
		},
		{
			name:  "full state",
			state: testState.InnerStateUnsafe(),
			error: "",
		},
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := state.InitializeFromProto(tt.state)
			if err != nil && err.Error() != tt.error {
				t.Errorf("Unexpected error, expected %v, recevied %v", tt.error, err)
			}
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
		})
	}
}

func TestInitializeFromProtoUnsafe(t *testing.T) {
	testState, _ := testutil.DeterministicGenesisState(t, 64)

	type test struct {
		name  string
		state *pbp2p.BeaconState
		error string
	}
	initTests := []test{
		{
			name:  "nil state",
			state: nil,
			error: "received nil state",
		},
		{
			name: "nil validators",
			state: &pbp2p.BeaconState{
				Slot:       4,
				Validators: nil,
			},
			error: "",
		},
		{
			name:  "empty state",
			state: &pbp2p.BeaconState{},
			error: "",
		},
		{
			name:  "full state",
			state: testState.InnerStateUnsafe(),
			error: "",
		},
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := state.InitializeFromProtoUnsafe(tt.state)
			if err != nil && err.Error() != tt.error {
				t.Errorf("Unexpected error, expected %v, recevied %v", tt.error, err)
			}
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
		})
	}
}

func TestBeaconState_HashTreeRoot(t *testing.T) {
	testState, _ := testutil.DeterministicGenesisState(t, 64)

	type test struct {
		name        string
		stateModify func(*state.BeaconState) (*state.BeaconState, error)
		error       string
	}
	initTests := []test{
		{
			name: "unchanged state",
			stateModify: func(beaconState *state.BeaconState) (*state.BeaconState, error) {
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different slot",
			stateModify: func(beaconState *state.BeaconState) (*state.BeaconState, error) {
				if err := beaconState.SetSlot(5); err != nil {
					return nil, err
				}
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different validator balance",
			stateModify: func(beaconState *state.BeaconState) (*state.BeaconState, error) {
				val, err := beaconState.ValidatorAtIndex(5)
				if err != nil {
					return nil, err
				}
				val.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement
				if err := beaconState.UpdateValidatorAtIndex(5, val); err != nil {
					return nil, err
				}
				return beaconState, nil
			},
			error: "",
		},
	}

	var err error
	var oldHTR []byte
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			testState, err = tt.stateModify(testState)
			if err != nil {
				t.Errorf("Unexpected error, expected %v, recevied %v", tt.error, err)
			}
			root, err := testState.HashTreeRoot(context.Background())
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
			genericHTR, err := ssz.HashTreeRoot(testState.InnerStateUnsafe())
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
			if bytes.Equal(root[:], []byte{}) {
				t.Error("Received empty hash tree root")
			}
			if !bytes.Equal(root[:], genericHTR[:]) {
				t.Error("Expected hash tree root to match generic")
			}
			if len(oldHTR) != 0 && bytes.Equal(root[:], oldHTR) {
				t.Errorf("Expected HTR to change, received %#x == old %#x", root, oldHTR)
			}
			oldHTR = root[:]
		})
	}
}

func TestBeaconState_HashTreeRoot_FieldTrie(t *testing.T) {
	testState, _ := testutil.DeterministicGenesisState(t, 64)

	type test struct {
		name        string
		stateModify func(*state.BeaconState) (*state.BeaconState, error)
		error       string
	}
	initTests := []test{
		{
			name: "unchanged state",
			stateModify: func(beaconState *state.BeaconState) (*state.BeaconState, error) {
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different slot",
			stateModify: func(beaconState *state.BeaconState) (*state.BeaconState, error) {
				if err := beaconState.SetSlot(5); err != nil {
					return nil, err
				}
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different validator balance",
			stateModify: func(beaconState *state.BeaconState) (*state.BeaconState, error) {
				val, err := beaconState.ValidatorAtIndex(5)
				if err != nil {
					return nil, err
				}
				val.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement
				if err := beaconState.UpdateValidatorAtIndex(5, val); err != nil {
					return nil, err
				}
				return beaconState, nil
			},
			error: "",
		},
	}

	var err error
	var oldHTR []byte
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			testState, err = tt.stateModify(testState)
			if err != nil {
				t.Errorf("Unexpected error, expected %v, recevied %v", tt.error, err)
			}
			root, err := testState.HashTreeRoot(context.Background())
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
			genericHTR, err := ssz.HashTreeRoot(testState.InnerStateUnsafe())
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
			if bytes.Equal(root[:], []byte{}) {
				t.Error("Received empty hash tree root")
			}
			if !bytes.Equal(root[:], genericHTR[:]) {
				t.Error("Expected hash tree root to match generic")
			}
			if len(oldHTR) != 0 && bytes.Equal(root[:], oldHTR) {
				t.Errorf("Expected HTR to change, received %#x == old %#x", root, oldHTR)
			}
			oldHTR = root[:]
		})
	}
}
