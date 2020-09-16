package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"

	"github.com/prysmaticlabs/prysm/shared/params"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestNewAttestationHistoryArray(t *testing.T) {
	ba := newAttestationHistoryArray(0)
	assert.Equal(t, latestEpochWrittenSize+historySize, len(ba))
	ba = newAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod - 1)
	assert.Equal(t, latestEpochWrittenSize+historySize*params.BeaconConfig().WeakSubjectivityPeriod, uint64(len(ba)))
	ba = newAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod)
	assert.Equal(t, latestEpochWrittenSize+historySize, len(ba))
	ba = newAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod + 1)
	assert.Equal(t, latestEpochWrittenSize+historySize+historySize, len(ba))

}

func TestSizeChecks(t *testing.T) {
	require.ErrorContains(t, "is smaller then minimal size", sizeChecks([]byte{}))
	require.NoError(t, sizeChecks([]byte{0, 1, 2, 3, 4, 5, 6, 7}))
	require.ErrorContains(t, "is not a multiple of entry size", sizeChecks([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8}))
	require.NoError(t, sizeChecks(newAttestationHistoryArray(0)))
	require.NoError(t, sizeChecks(newAttestationHistoryArray(1)))
	require.NoError(t, sizeChecks(newAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod)))
	require.NoError(t, sizeChecks(newAttestationHistoryArray(params.BeaconConfig().WeakSubjectivityPeriod-1)))
}

func TestGetLatestEpochWritten(t *testing.T) {
	ctx := context.Background()
	ha := newAttestationHistoryArray(0)
	ha[0] = 28
	lew, err := getLatestEpochWritten(ctx, ha)
	require.NoError(t, err)
	assert.Equal(t, uint64(28), lew)
}

func TestSetLatestEpochWritten(t *testing.T) {
	ctx := context.Background()
	ha := newAttestationHistoryArray(0)
	lew, err := setLatestEpochWritten(ctx, ha, 2828282828)
	require.NoError(t, err)
	assert.Equal(t, uint64(2828282828), bytesutil.FromBytes8(lew[:latestEpochWrittenSize]))
}

func TestGetTargetData(t *testing.T) {
	ctx := context.Background()
	ha := newAttestationHistoryArray(0)
	td, err := getTargetData(ctx, ha, 0)
	require.NoError(t, err)
	assert.DeepEqual(t, &HistoryData{
		Source:      0,
		SigningRoot: bytesutil.PadTo([]byte{}, 32),
	}, td)
	td, err = getTargetData(ctx, ha, 1)
	require.ErrorContains(t, "is smaller then the requested target location", err)
}

func TestAttestationHistoryForPubKeysNew_EmptyVals(t *testing.T) {
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := setupDB(t, pubkeys)

	historyForPubKeys, err := db.AttestationHistoryNewForPubKeys(context.Background(), pubkeys)
	require.NoError(t, err)

	cleanAttHistoryForPubKeys := make(map[[48]byte][]byte)
	clean := make([]byte, latestEpochWrittenSize)
	for _, pubKey := range pubkeys {
		cleanAttHistoryForPubKeys[pubKey] = clean
	}

	require.DeepEqual(t, cleanAttHistoryForPubKeys, historyForPubKeys, "Expected attestation history epoch bits to be empty")
}

//func TestSaveAttestationHistory_OK(t *testing.T) {
//	pubKeys := [][48]byte{{3}, {4}}
//	db := setupDB(t, pubKeys)
//
//	farFuture := params.BeaconConfig().FarFutureEpoch
//	newMap := make(map[uint64]uint64)
//	// The validator attested at target epoch 2 but had no attestations for target epochs 0 and 1.
//	newMap[0] = farFuture
//	newMap[1] = farFuture
//	newMap[2] = 1
//	history := &slashpb.AttestationHistory{
//		TargetToSource:     newMap,
//		LatestEpochWritten: 2,
//	}
//
//	newMap2 := make(map[uint64]uint64)
//	// The validator attested at target epoch 1 and 3 but had no attestations for target epochs 0 and 2.
//	newMap2[0] = farFuture
//	newMap2[1] = 0
//	newMap2[2] = farFuture
//	newMap2[3] = 2
//	history2 := &slashpb.AttestationHistory{
//		TargetToSource:     newMap2,
//		LatestEpochWritten: 3,
//	}
//
//	attestationHistory := make(map[[48]byte]*slashpb.AttestationHistory)
//	attestationHistory[pubKeys[0]] = history
//	attestationHistory[pubKeys[1]] = history2
//
//	require.NoError(t, db.SaveAttestationHistoryForPubKeys(context.Background(), attestationHistory), "Saving attestation history failed")
//	savedHistories, err := db.AttestationHistoryForPubKeys(context.Background(), pubKeys)
//	require.NoError(t, err, "Failed to get attestation history")
//
//	require.NotNil(t, savedHistories)
//	require.DeepEqual(t, attestationHistory, savedHistories, "Expected DB to keep object the same, received: %v", history)
//
//	savedHistory := savedHistories[pubKeys[0]]
//	require.Equal(t, newMap[2], savedHistory.TargetToSource[2], "Expected target epoch %d to have the same marked source epoch", 2)
//	require.Equal(t, newMap[1], savedHistory.TargetToSource[1], "Expected target epoch %d to have the same marked source epoch", 1)
//	require.Equal(t, newMap[0], savedHistory.TargetToSource[0], "Expected target epoch %d to have the same marked source epoch", 0)
//
//	savedHistory = savedHistories[pubKeys[1]]
//	require.Equal(t, newMap2[3], savedHistory.TargetToSource[3], "Expected target epoch %d to have the same marked source epoch", 3)
//	require.Equal(t, newMap2[2], savedHistory.TargetToSource[2], "Expected target epoch %d to have the same marked source epoch", 2)
//	require.Equal(t, newMap2[1], savedHistory.TargetToSource[1], "Expected target epoch %d to have the same marked source epoch", 1)
//}
//
//func TestSaveAttestationHistory_Overwrites(t *testing.T) {
//	db := setupDB(t, [][48]byte{})
//	farFuture := params.BeaconConfig().FarFutureEpoch
//	newMap1 := make(map[uint64]uint64)
//	newMap1[0] = farFuture
//	newMap1[1] = 0
//	newMap2 := make(map[uint64]uint64)
//	newMap2[0] = farFuture
//	newMap2[1] = farFuture
//	newMap2[2] = 1
//	newMap3 := make(map[uint64]uint64)
//	newMap3[0] = farFuture
//	newMap3[1] = farFuture
//	newMap3[2] = farFuture
//	newMap3[3] = 2
//	tests := []struct {
//		pubkey  [48]byte
//		epoch   uint64
//		history *slashpb.AttestationHistory
//	}{
//		{
//			pubkey: [48]byte{0},
//			epoch:  uint64(1),
//			history: &slashpb.AttestationHistory{
//				TargetToSource:     newMap1,
//				LatestEpochWritten: 1,
//			},
//		},
//		{
//			pubkey: [48]byte{0},
//			epoch:  uint64(2),
//			history: &slashpb.AttestationHistory{
//				TargetToSource:     newMap2,
//				LatestEpochWritten: 2,
//			},
//		},
//		{
//			pubkey: [48]byte{0},
//			epoch:  uint64(3),
//			history: &slashpb.AttestationHistory{
//				TargetToSource:     newMap3,
//				LatestEpochWritten: 3,
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		attHistory := make(map[[48]byte]*slashpb.AttestationHistory)
//		attHistory[tt.pubkey] = tt.history
//		require.NoError(t, db.SaveAttestationHistoryForPubKeys(context.Background(), attHistory), "Saving attestation history failed")
//		histories, err := db.AttestationHistoryForPubKeys(context.Background(), [][48]byte{tt.pubkey})
//		require.NoError(t, err, "Failed to get attestation history")
//
//		history := histories[tt.pubkey]
//		require.NotNil(t, history)
//		require.DeepEqual(t, tt.history, history, "Expected DB to keep object the same")
//		require.Equal(t, tt.epoch-1, history.TargetToSource[tt.epoch],
//			"Expected target epoch %d to be marked with correct source epoch %d", tt.epoch, history.TargetToSource[tt.epoch])
//		require.Equal(t, farFuture, history.TargetToSource[tt.epoch-1],
//			"Expected target epoch %d to not be marked as attested for, received %d", tt.epoch-1, history.TargetToSource[tt.epoch-1])
//	}
//}
