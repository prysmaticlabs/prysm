package cache_test

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestSyncCommitteeCache_CanUpdateAndRetrieve(t *testing.T) {
	numValidators := 101
	deterministicState, _ := util.DeterministicGenesisStateAltair(t, uint64(numValidators))
	pubKeys := make([][]byte, deterministicState.NumValidators())
	for i, val := range deterministicState.Validators() {
		pubKeys[i] = val.PublicKey
	}
	tests := []struct {
		name                 string
		currentSyncCommittee *ethpb.SyncCommittee
		nextSyncCommittee    *ethpb.SyncCommittee
		currentSyncMap       map[types.ValidatorIndex][]types.CommitteeIndex
		nextSyncMap          map[types.ValidatorIndex][]types.CommitteeIndex
	}{
		{
			name: "only current epoch",
			currentSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[1], pubKeys[2], pubKeys[3], pubKeys[2], pubKeys[2],
			}),
			nextSyncCommittee: util.ConvertToCommittee([][]byte{}),
			currentSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				1: {0},
				2: {1, 3, 4},
				3: {2},
			},
			nextSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				1: {},
				2: {},
				3: {},
			},
		},
		{
			name:                 "only next epoch",
			currentSyncCommittee: util.ConvertToCommittee([][]byte{}),
			nextSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[1], pubKeys[2], pubKeys[3], pubKeys[2], pubKeys[2],
			}),
			currentSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				1: {},
				2: {},
				3: {},
			},
			nextSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				1: {0},
				2: {1, 3, 4},
				3: {2},
			},
		},
		{
			name: "some current epoch and some next epoch",
			currentSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[1],
				pubKeys[2],
				pubKeys[3],
				pubKeys[2],
				pubKeys[2],
			}),
			nextSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[7],
				pubKeys[6],
				pubKeys[5],
				pubKeys[4],
				pubKeys[7],
			}),
			currentSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				1: {0},
				2: {1, 3, 4},
				3: {2},
			},
			nextSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				7: {0, 4},
				6: {1},
				5: {2},
				4: {3},
			},
		},
		{
			name: "some current epoch and some next epoch duplicated across",
			currentSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[1],
				pubKeys[2],
				pubKeys[3],
				pubKeys[2],
				pubKeys[2],
			}),
			nextSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[2],
				pubKeys[1],
				pubKeys[3],
				pubKeys[2],
				pubKeys[1],
			}),
			currentSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				1: {0},
				2: {1, 3, 4},
				3: {2},
			},
			nextSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				1: {1, 4},
				2: {0, 3},
				3: {2},
			},
		},
		{
			name: "all duplicated",
			currentSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[100],
				pubKeys[100],
				pubKeys[100],
				pubKeys[100],
			}),
			nextSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[100],
				pubKeys[100],
				pubKeys[100],
				pubKeys[100],
			}),
			currentSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				100: {0, 1, 2, 3},
			},
			nextSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				100: {0, 1, 2, 3},
			},
		},
		{
			name: "unknown keys",
			currentSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[100],
				pubKeys[100],
				pubKeys[100],
				pubKeys[100],
			}),
			nextSyncCommittee: util.ConvertToCommittee([][]byte{
				pubKeys[100],
				pubKeys[100],
				pubKeys[100],
				pubKeys[100],
			}),
			currentSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				1: {},
			},
			nextSyncMap: map[types.ValidatorIndex][]types.CommitteeIndex{
				1: {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := util.DeterministicGenesisStateAltair(t, uint64(numValidators))
			require.NoError(t, s.SetCurrentSyncCommittee(tt.currentSyncCommittee))
			require.NoError(t, s.SetNextSyncCommittee(tt.nextSyncCommittee))
			cache := cache.NewSyncCommittee()
			r := [32]byte{'a'}
			require.NoError(t, cache.UpdatePositionsInCommittee(r, s))
			for key, indices := range tt.currentSyncMap {
				pos, err := cache.CurrentPeriodIndexPosition(r, key)
				require.NoError(t, err)
				require.DeepEqual(t, indices, pos)
			}
			for key, indices := range tt.nextSyncMap {
				pos, err := cache.NextPeriodIndexPosition(r, key)
				require.NoError(t, err)
				require.DeepEqual(t, indices, pos)
			}
		})
	}
}

func TestSyncCommitteeCache_RootDoesNotExist(t *testing.T) {
	c := cache.NewSyncCommittee()
	_, err := c.CurrentPeriodIndexPosition([32]byte{}, 0)
	require.Equal(t, cache.ErrNonExistingSyncCommitteeKey, err)
}

func TestSyncCommitteeCache_CanRotate(t *testing.T) {
	c := cache.NewSyncCommittee()
	s, _ := util.DeterministicGenesisStateAltair(t, 64)
	require.NoError(t, s.SetCurrentSyncCommittee(util.ConvertToCommittee([][]byte{{1}})))
	require.NoError(t, c.UpdatePositionsInCommittee([32]byte{'a'}, s))
	require.NoError(t, s.SetCurrentSyncCommittee(util.ConvertToCommittee([][]byte{{2}})))
	require.NoError(t, c.UpdatePositionsInCommittee([32]byte{'b'}, s))
	require.NoError(t, s.SetCurrentSyncCommittee(util.ConvertToCommittee([][]byte{{3}})))
	require.NoError(t, c.UpdatePositionsInCommittee([32]byte{'c'}, s))
	require.NoError(t, s.SetCurrentSyncCommittee(util.ConvertToCommittee([][]byte{{4}})))
	require.NoError(t, c.UpdatePositionsInCommittee([32]byte{'d'}, s))

	_, err := c.CurrentPeriodIndexPosition([32]byte{'a'}, 0)
	require.Equal(t, cache.ErrNonExistingSyncCommitteeKey, err)

	_, err = c.CurrentPeriodIndexPosition([32]byte{'c'}, 0)
	require.NoError(t, err)
}
