package v2_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"google.golang.org/protobuf/proto"
)

func TestInitializeFromProto(t *testing.T) {
	testState, _ := util.DeterministicGenesisStateAltair(t, 64)
	pbState, err := stateAltair.ProtobufBeaconState(testState.InnerStateUnsafe())
	require.NoError(t, err)
	type test struct {
		name  string
		state *ethpb.BeaconStateAltair
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
			state: &ethpb.BeaconStateAltair{
				Slot:       4,
				Validators: nil,
			},
		},
		{
			name:  "empty state",
			state: &ethpb.BeaconStateAltair{},
		},
		{
			name:  "full state",
			state: pbState,
		},
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := stateAltair.InitializeFromProto(tt.state)
			if tt.error != "" {
				require.ErrorContains(t, tt.error, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInitializeFromProtoUnsafe(t *testing.T) {
	testState, _ := util.DeterministicGenesisStateAltair(t, 64)
	pbState, err := stateAltair.ProtobufBeaconState(testState.InnerStateUnsafe())
	require.NoError(t, err)
	type test struct {
		name  string
		state *ethpb.BeaconStateAltair
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
			state: &ethpb.BeaconStateAltair{
				Slot:       4,
				Validators: nil,
			},
		},
		{
			name:  "empty state",
			state: &ethpb.BeaconStateAltair{},
		},
		{
			name:  "full state",
			state: pbState,
		},
	}
	for _, tt := range initTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := stateAltair.InitializeFromProtoUnsafe(tt.state)
			if tt.error != "" {
				assert.ErrorContains(t, tt.error, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBeaconState_HashTreeRoot(t *testing.T) {
	t.Skip("Failing test pending resolution https://github.com/ferranbt/fastssz/issues/67")
	testState, _ := util.DeterministicGenesisStateAltair(t, 64)
	type test struct {
		name        string
		stateModify func(beaconState state.BeaconStateAltair) (state.BeaconStateAltair, error)
		error       string
	}
	initTests := []test{
		{
			name: "unchanged state",
			stateModify: func(beaconState state.BeaconStateAltair) (state.BeaconStateAltair, error) {
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different slot",
			stateModify: func(beaconState state.BeaconStateAltair) (state.BeaconStateAltair, error) {
				if err := beaconState.SetSlot(5); err != nil {
					return nil, err
				}
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different validator balance",
			stateModify: func(beaconState state.BeaconStateAltair) (state.BeaconStateAltair, error) {
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
			pbState, err := stateAltair.ProtobufBeaconState(testState.InnerStateUnsafe())
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
	t.Skip("Failing test pending resolution https://github.com/ferranbt/fastssz/issues/67")
	testState, _ := util.DeterministicGenesisStateAltair(t, 64)

	type test struct {
		name        string
		stateModify func(state.BeaconStateAltair) (state.BeaconStateAltair, error)
		error       string
	}
	initTests := []test{
		{
			name: "unchanged state",
			stateModify: func(beaconState state.BeaconStateAltair) (state.BeaconStateAltair, error) {
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different slot",
			stateModify: func(beaconState state.BeaconStateAltair) (state.BeaconStateAltair, error) {
				if err := beaconState.SetSlot(5); err != nil {
					return nil, err
				}
				return beaconState, nil
			},
			error: "",
		},
		{
			name: "different validator balance",
			stateModify: func(beaconState state.BeaconStateAltair) (state.BeaconStateAltair, error) {
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
			pbState, err := stateAltair.ProtobufBeaconState(testState.InnerStateUnsafe())
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

func TestBeaconStateAltair_ProtoBeaconStateCompatibility(t *testing.T) {
	ctx := context.Background()
	s, _ := util.DeterministicGenesisStateAltair(t, 6)
	inner := s.InnerStateUnsafe()
	genesis, err := stateAltair.ProtobufBeaconState(inner)
	require.NoError(t, err)
	customState, err := stateAltair.InitializeFromProto(genesis)
	require.NoError(t, err)
	cloned, ok := proto.Clone(genesis).(*ethpb.BeaconStateAltair)
	assert.Equal(t, true, ok, "Object is not of type *ethpb.BeaconStateAltair")
	custom := customState.CloneInnerState()
	assert.DeepSSZEqual(t, cloned, custom)
	r1, err := customState.HashTreeRoot(ctx)
	require.NoError(t, err)
	beaconState, err := stateAltair.InitializeFromProto(genesis)
	require.NoError(t, err)
	r2, err := beaconState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, r1, r2, "Mismatched roots")

	// We then write to the the state and compare hash tree roots again.
	balances := genesis.Balances
	balances[0] = 3823
	require.NoError(t, customState.SetBalances(balances))
	r1, err = customState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesis.Balances = balances
	beaconState, err = stateAltair.InitializeFromProto(genesis)
	require.NoError(t, err)
	r2, err = beaconState.HashTreeRoot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, r1, r2, "Mismatched roots")
}
