package monitor

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestTrackedIndex(t *testing.T) {
	s := &Service{
		config: &ValidatorMonitorConfig{
			TrackedValidators: map[types.ValidatorIndex]interface{}{
				1: nil,
				2: nil,
			},
		},
	}
	require.Equal(t, s.trackedIndex(types.ValidatorIndex(1)), true)
	require.Equal(t, s.trackedIndex(types.ValidatorIndex(3)), false)
}

func TestUpdateSyncCommitteeTrackedVals(t *testing.T) {
	hook := logTest.NewGlobal()
	s := setupService(t)
	state, _ := util.DeterministicGenesisStateAltair(t, 1024)

	pubKeys := make([][]byte, 3)
	pubKeys[0] = state.Validators()[0].PublicKey
	pubKeys[1] = state.Validators()[1].PublicKey
	pubKeys[2] = state.Validators()[2].PublicKey

	currentSyncCommittee := util.ConvertToCommittee([][]byte{
		pubKeys[0], pubKeys[1], pubKeys[2], pubKeys[1], pubKeys[1],
	})
	require.NoError(t, state.SetCurrentSyncCommittee(currentSyncCommittee))

	s.updateSyncCommitteeTrackedVals(state)
	require.LogsDoNotContain(t, hook, "Sync committee assignments will not be reported")
	newTrackedSyncIndices := map[types.ValidatorIndex][]types.CommitteeIndex{
		1: {1, 3, 4},
		2: {2},
	}
	require.DeepEqual(t, s.trackedSyncCommitteeIndices, newTrackedSyncIndices)
}
