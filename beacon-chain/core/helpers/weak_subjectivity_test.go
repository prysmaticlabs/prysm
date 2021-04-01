package helpers

import (
	"fmt"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateV0"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestWeakSubjectivity_ComputeWeakSubjectivityPeriod(t *testing.T) {
	genState := func(valCount uint64, avgBalance uint64) iface.ReadOnlyBeaconState {
		registry := make([]*ethpb.Validator, valCount)
		for i := uint64(0); i < valCount; i++ {
			registry[i] = &ethpb.Validator{
				EffectiveBalance: avgBalance * 1e9,
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			}
		}
		beaconState, err := stateV0.InitializeFromProto(&pb.BeaconState{
			Validators: registry,
			Slot:       200,
		})
		require.NoError(t, err)

		return beaconState
	}
	tests := []struct {
		valCount   uint64
		avgBalance uint64
		want       types.Epoch
	}{
		// Asserting that we get the same numbers as defined in the reference table:
		// https://github.com/ethereum/eth2.0-specs/blob/master/specs/phase0/weak-subjectivity.md#calculating-the-weak-subjectivity-period
		{valCount: 32768, avgBalance: 28, want: 504},
		{valCount: 65536, avgBalance: 28, want: 752},
		{valCount: 131072, avgBalance: 28, want: 1248},
		{valCount: 262144, avgBalance: 28, want: 2241},
		{valCount: 524288, avgBalance: 28, want: 2241},
		{valCount: 1048576, avgBalance: 28, want: 2241},
		{valCount: 32768, avgBalance: 32, want: 665},
		{valCount: 65536, avgBalance: 32, want: 1075},
		{valCount: 131072, avgBalance: 32, want: 1894},
		{valCount: 262144, avgBalance: 32, want: 3532},
		{valCount: 524288, avgBalance: 32, want: 3532},
		{valCount: 1048576, avgBalance: 32, want: 3532},
		// Additional test vectors, to check case when T*(200+3*D) >= t*(200+12*D)
		{valCount: 32768, avgBalance: 22, want: 277},
		{valCount: 65536, avgBalance: 22, want: 298},
		{valCount: 131072, avgBalance: 22, want: 340},
		{valCount: 262144, avgBalance: 22, want: 424},
		{valCount: 524288, avgBalance: 22, want: 593},
		{valCount: 1048576, avgBalance: 22, want: 931},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("valCount: %d, avgBalance: %d", tt.valCount, tt.avgBalance), func(t *testing.T) {
			// Reset committee cache - as we need to recalculate active validator set for each test.
			committeeCache = cache.NewCommitteesCache()
			got, err := ComputeWeakSubjectivityPeriod(genState(tt.valCount, tt.avgBalance))
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "valCount: %v, avgBalance: %v", tt.valCount, tt.avgBalance)
		})
	}
}
