package blocks_test

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProcessAttesterSlashings_RegressionSlashableIndices(t *testing.T) {
	testutil.ResetCache()
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 5500)
	for _, vv := range beaconState.Validators() {
		vv.WithdrawableEpoch = 1 * params.BeaconConfig().SlotsPerEpoch
	}
	// This set of indices is very similar to the one from our sapphire testnet
	// when close to 100 validators were incorrectly slashed. The set is from 0 -5500,
	// instead of 55000 as it would take too long to generate a state.
	setA := []uint64{21, 92, 236, 244, 281, 321, 510, 524,
		538, 682, 828, 858, 913, 920, 922, 959, 1176, 1207,
		1222, 1229, 1354, 1394, 1436, 1454, 1510, 1550,
		1552, 1576, 1645, 1704, 1842, 1967, 2076, 2111, 2134, 2307,
		2343, 2354, 2417, 2524, 2532, 2555, 2740, 2749, 2759, 2762,
		2800, 2809, 2824, 2987, 3110, 3125, 3559, 3583, 3599, 3608,
		3657, 3685, 3723, 3756, 3759, 3761, 3820, 3826, 3979, 4030,
		4141, 4170, 4205, 4247, 4257, 4479, 4492, 4569, 5091,
	}
	// Only 2800 is the slashable index.
	setB := []uint64{1361, 1438, 2383, 2800}
	expectedSlashedVal := 2800

	root1 := [32]byte{'d', 'o', 'u', 'b', 'l', 'e', '1'}
	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: root1[:]},
		},
		AttestingIndices: setA,
	}
	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	require.NoError(t, err, "Could not get signing root of beacon block header")
	aggSigs := []bls.Signature{}
	for _, index := range setA {
		sig := privKeys[index].Sign(signingRoot[:])
		aggSigs = append(aggSigs, sig)
	}
	aggregateSig := bls.AggregateSignatures(aggSigs)
	att1.Signature = aggregateSig.Marshal()[:]

	root2 := [32]byte{'d', 'o', 'u', 'b', 'l', 'e', '2'}
	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: root2[:]},
		},
		AttestingIndices: setB,
	}
	signingRoot, err = helpers.ComputeSigningRoot(att2.Data, domain)
	assert.NoError(t, err, "Could not get signing root of beacon block header")
	aggSigs = []bls.Signature{}
	for _, index := range setB {
		sig := privKeys[index].Sign(signingRoot[:])
		aggSigs = append(aggSigs, sig)
	}
	aggregateSig = bls.AggregateSignatures(aggSigs)
	att2.Signature = aggregateSig.Marshal()[:]

	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}

	currentSlot := 2 * params.BeaconConfig().SlotsPerEpoch
	require.NoError(t, beaconState.SetSlot(currentSlot))

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}

	newState, err := blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body)
	require.NoError(t, err)
	newRegistry := newState.Validators()
	if !newRegistry[expectedSlashedVal].Slashed {
		t.Errorf("Validator with index %d was not slashed despite performing a double vote", expectedSlashedVal)
	}

	for idx, val := range newRegistry {
		if val.Slashed && idx != expectedSlashedVal {
			t.Errorf("validator with index: %d was unintentionally slashed", idx)
		}
	}
}
