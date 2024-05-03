package state_native_test

import (
	"testing"

	"github.com/golang/snappy"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestExitEpochAndUpdateChurn(t *testing.T) {
	// Load a serialized Electra state from disk.
	// The spec tests shows that the exit epoch is 262 for validator 0 performing a voluntary exit.
	serializedBytes, err := util.BazelFileBytes("tests/mainnet/electra/operations/voluntary_exit/pyspec_tests/exit_existing_churn_and_churn_limit_balance/pre.ssz_snappy")
	require.NoError(t, err)
	serializedSSZ, err := snappy.Decode(nil /* dst */, serializedBytes)
	require.NoError(t, err)
	pb := &eth.BeaconStateElectra{}
	require.NoError(t, pb.UnmarshalSSZ(serializedSSZ))
	s, err := state_native.InitializeFromProtoElectra(pb)
	require.NoError(t, err)

	val, err := s.ValidatorAtIndex(0)
	require.NoError(t, err)

	ee, err := s.ExitEpochAndUpdateChurn(val.EffectiveBalance)
	require.NoError(t, err)
	require.Equal(t, primitives.Epoch(262), ee)

	// Fails for versions older than electra
	s, err = state_native.InitializeFromProtoDeneb(&eth.BeaconStateDeneb{})
	require.NoError(t, err)
	_, err = s.ExitEpochAndUpdateChurn(10)
	require.ErrorContains(t, "not supported", err)
}
