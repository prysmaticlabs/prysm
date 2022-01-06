package v1_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestMain(m *testing.M) {
	resetCfg := features.InitWithReset(&features.Flags{EnableBalanceTrieComputation: true})
	defer resetCfg()
	m.Run()
}

func TestInitializeFromProto(t *testing.T) {
	testState, _ := util.DeterministicGenesisState(t, 64)
	pbState, err := v1.ProtobufBeaconState(testState.InnerStateUnsafe())
	require.NoError(t, err)
	type test struct {
		name  string
		state *ethpb.BeaconState
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
			state: &ethpb.BeaconState{
				Slot:       4,
				Validators: nil,
			},
		},
		{
			name:  "empty state",
			state: &ethpb.BeaconState{},
		},
		{
			name:  "full state",
			state: pbState,
		},
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v1.InitializeFromProto(tt.state)
			if tt.error != "" {
				assert.ErrorContains(t, tt.error, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitializeFromProtoUnsafe(t *testing.T) {
	testState, _ := util.DeterministicGenesisState(t, 64)
	pbState, err := v1.ProtobufBeaconState(testState.InnerStateUnsafe())
	require.NoError(t, err)
	type test struct {
		name  string
		state *ethpb.BeaconState
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
			state: &ethpb.BeaconState{
				Slot:       4,
				Validators: nil,
			},
		},
		{
			name:  "empty state",
			state: &ethpb.BeaconState{},
		},
		{
			name:  "full state",
			state: pbState,
		},
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v1.InitializeFromProtoUnsafe(tt.state)
			if tt.error != "" {
				assert.ErrorContains(t, tt.error, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBeaconState_HashTreeRoot(t *testing.T) {
	testState, _ := util.DeterministicGenesisState(t, 64)

	type test struct {
		name        string
		stateModify func(beaconState state.BeaconState) (state.BeaconState, error)
		error       string
	}
	initTests := []test{
		{
			name: "unchanged state",
			stateModify: func(beaconState state.BeaconState) (state.BeaconState, error) {
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different slot",
			stateModify: func(beaconState state.BeaconState) (state.BeaconState, error) {
				if err := beaconState.SetSlot(5); err != nil {
					return nil, err
				}
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different validator balance",
			stateModify: func(beaconState state.BeaconState) (state.BeaconState, error) {
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
			assert.NoError(t, err)
			root, err := testState.HashTreeRoot(context.Background())
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
			pbState, err := v1.ProtobufBeaconState(testState.InnerStateUnsafe())
			require.NoError(t, err)
			genericHTR, err := pbState.HashTreeRoot()
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
			assert.DeepNotEqual(t, []byte{}, root[:], "Received empty hash tree root")
			assert.DeepEqual(t, genericHTR[:], root[:], "Expected hash tree root to match generic")
			if len(oldHTR) != 0 && bytes.Equal(root[:], oldHTR) {
				t.Errorf("Expected HTR to change, received %#x == old %#x", root, oldHTR)
			}
			oldHTR = root[:]
		})
	}
}

func TestBeaconState_HashTreeRoot_FieldTrie(t *testing.T) {
	testState, _ := util.DeterministicGenesisState(t, 64)

	type test struct {
		name        string
		stateModify func(state.BeaconState) (state.BeaconState, error)
		error       string
	}
	initTests := []test{
		{
			name: "unchanged state",
			stateModify: func(beaconState state.BeaconState) (state.BeaconState, error) {
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different slot",
			stateModify: func(beaconState state.BeaconState) (state.BeaconState, error) {
				if err := beaconState.SetSlot(5); err != nil {
					return nil, err
				}
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different validator balance",
			stateModify: func(beaconState state.BeaconState) (state.BeaconState, error) {
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
			assert.NoError(t, err)
			root, err := testState.HashTreeRoot(context.Background())
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
			pbState, err := v1.ProtobufBeaconState(testState.InnerStateUnsafe())
			require.NoError(t, err)
			genericHTR, err := pbState.HashTreeRoot()
			if err == nil && tt.error != "" {
				t.Errorf("Expected error, expected %v, recevied %v", tt.error, err)
			}
			assert.DeepNotEqual(t, []byte{}, root[:], "Received empty hash tree root")
			assert.DeepEqual(t, genericHTR[:], root[:], "Expected hash tree root to match generic")
			if len(oldHTR) != 0 && bytes.Equal(root[:], oldHTR) {
				t.Errorf("Expected HTR to change, received %#x == old %#x", root, oldHTR)
			}
			oldHTR = root[:]
		})
	}
}

func TestBeaconState_AppendValidator_DoesntMutateCopy(t *testing.T) {
	st0, err := util.NewBeaconState()
	require.NoError(t, err)
	st1 := st0.Copy()
	originalCount := st1.NumValidators()

	val := &ethpb.Validator{Slashed: true}
	assert.NoError(t, st0.AppendValidator(val))
	assert.Equal(t, originalCount, st1.NumValidators(), "st1 NumValidators mutated")
	_, ok := st1.ValidatorIndexByPubkey(bytesutil.ToBytes48(val.PublicKey))
	assert.Equal(t, false, ok, "Expected no validator index to be present in st1 for the newly inserted pubkey")
}
