package cache_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestSyncCommitteeCache_CanUpdateAndRetrieve(t *testing.T) {
	tests := []struct {
		name                 string
		currentSyncCommittee *pb.SyncCommittee
		nextSyncCommittee    *pb.SyncCommittee
		currentSyncMap       map[[48]byte][]uint64
		nextSyncMap          map[[48]byte][]uint64
	}{
		{
			name:                 "only current epoch",
			currentSyncCommittee: convertToCommittee([][]byte{{1}, {2}, {3}, {2}, {2}}),
			nextSyncCommittee:    convertToCommittee([][]byte{}),
			currentSyncMap: map[[48]byte][]uint64{
				[48]byte{1}: {0},
				[48]byte{2}: {1, 3, 4},
				[48]byte{3}: {2},
			},
			nextSyncMap: map[[48]byte][]uint64{
				[48]byte{1}: {},
				[48]byte{2}: {},
				[48]byte{3}: {},
			},
		},
		{
			name:                 "only next epoch",
			currentSyncCommittee: convertToCommittee([][]byte{}),
			nextSyncCommittee:    convertToCommittee([][]byte{{1}, {2}, {3}, {2}, {2}}),
			currentSyncMap: map[[48]byte][]uint64{
				[48]byte{1}: {},
				[48]byte{2}: {},
				[48]byte{3}: {},
			},
			nextSyncMap: map[[48]byte][]uint64{
				[48]byte{1}: {0},
				[48]byte{2}: {1, 3, 4},
				[48]byte{3}: {2},
			},
		},
		{
			name:                 "some current epoch and some next epoch",
			currentSyncCommittee: convertToCommittee([][]byte{{1}, {2}, {3}, {2}, {2}}),
			nextSyncCommittee:    convertToCommittee([][]byte{{7}, {6}, {5}, {4}, {7}}),
			currentSyncMap: map[[48]byte][]uint64{
				[48]byte{1}: {0},
				[48]byte{2}: {1, 3, 4},
				[48]byte{3}: {2},
			},
			nextSyncMap: map[[48]byte][]uint64{
				[48]byte{7}: {0, 4},
				[48]byte{6}: {1},
				[48]byte{5}: {2},
				[48]byte{4}: {3},
			},
		},
		{
			name:                 "some current epoch and some next epoch duplicated across",
			currentSyncCommittee: convertToCommittee([][]byte{{1}, {2}, {3}, {2}, {2}}),
			nextSyncCommittee:    convertToCommittee([][]byte{{2}, {1}, {3}, {2}, {1}}),
			currentSyncMap: map[[48]byte][]uint64{
				[48]byte{1}: {0},
				[48]byte{2}: {1, 3, 4},
				[48]byte{3}: {2},
			},
			nextSyncMap: map[[48]byte][]uint64{
				[48]byte{1}: {1, 4},
				[48]byte{2}: {0, 3},
				[48]byte{3}: {2},
			},
		},
		{
			name:                 "all duplicated",
			currentSyncCommittee: convertToCommittee([][]byte{{100}, {100}, {100}, {100}}),
			nextSyncCommittee:    convertToCommittee([][]byte{{100}, {100}, {100}, {100}}),
			currentSyncMap: map[[48]byte][]uint64{
				[48]byte{100}: {0, 1, 2, 3},
			},
			nextSyncMap: map[[48]byte][]uint64{
				[48]byte{100}: {0, 1, 2, 3},
			},
		},
		{
			name:                 "unknown keys",
			currentSyncCommittee: convertToCommittee([][]byte{{100}, {100}, {100}, {100}}),
			nextSyncCommittee:    convertToCommittee([][]byte{{100}, {100}, {100}, {100}}),
			currentSyncMap: map[[48]byte][]uint64{
				[48]byte{1}: {},
			},
			nextSyncMap: map[[48]byte][]uint64{
				[48]byte{1}: {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := testutil.DeterministicGenesisStateAltair(t, 64)
			require.NoError(t, s.SetCurrentSyncCommittee(tt.currentSyncCommittee))
			require.NoError(t, s.SetNextSyncCommittee(tt.nextSyncCommittee))
			cache := cache.NewSyncCommittee()
			require.NoError(t, cache.UpdatePositionsInCommittee(s))
			csc, err := s.CurrentSyncCommittee()
			require.NoError(t, err)
			r, err := csc.HashTreeRoot()
			require.NoError(t, err)
			for key, indices := range tt.currentSyncMap {
				pos, err := cache.CurrentEpochIndexPosition(r, key)
				require.NoError(t, err)
				require.DeepEqual(t, indices, pos)
			}
			for key, indices := range tt.nextSyncMap {
				pos, err := cache.NextEpochIndexPosition(r, key)
				require.NoError(t, err)
				require.DeepEqual(t, indices, pos)
			}
		})
	}
}

func TestSyncCommitteeCache_RootDoesNotExist(t *testing.T) {
	c := cache.NewSyncCommittee()
	_, err := c.CurrentEpochIndexPosition([32]byte{}, [48]byte{})
	require.Equal(t, cache.ErrNonExistingSyncCommitteeKey, err)
}

func TestSyncCommitteeCache_CanRotate(t *testing.T) {
	c := cache.NewSyncCommittee()
	s, _ := testutil.DeterministicGenesisStateAltair(t, 64)
	require.NoError(t, s.SetCurrentSyncCommittee(convertToCommittee([][]byte{{1}})))
	require.NoError(t, c.UpdatePositionsInCommittee(s))

	csc, err := s.CurrentSyncCommittee()
	require.NoError(t, err)
	r, err := csc.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, s.SetCurrentSyncCommittee(convertToCommittee([][]byte{{2}})))
	require.NoError(t, c.UpdatePositionsInCommittee(s))
	require.NoError(t, s.SetCurrentSyncCommittee(convertToCommittee([][]byte{{3}})))
	require.NoError(t, c.UpdatePositionsInCommittee(s))
	require.NoError(t, s.SetCurrentSyncCommittee(convertToCommittee([][]byte{{4}})))
	require.NoError(t, c.UpdatePositionsInCommittee(s))

	_, err = c.CurrentEpochIndexPosition(r, [48]byte{})
	require.Equal(t, cache.ErrNonExistingSyncCommitteeKey, err)

	csc, err = s.CurrentSyncCommittee()
	require.NoError(t, err)
	r, err = csc.HashTreeRoot()
	require.NoError(t, err)
	_, err = c.CurrentEpochIndexPosition(r, [48]byte{})
	require.NoError(t, err)
}

func convertToCommittee(inputKeys [][]byte) *pb.SyncCommittee {
	var pubKeys [][]byte
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSize; i++ {
		if i < uint64(len(inputKeys)) {
			pubKeys = append(pubKeys, bytesutil.PadTo(inputKeys[i], params.BeaconConfig().BLSPubkeyLength))
		} else {
			pubKeys = append(pubKeys, bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength))
		}
	}

	return &pb.SyncCommittee{
		Pubkeys:         pubKeys,
		AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}
}
