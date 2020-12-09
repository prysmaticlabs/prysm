package interchangeformat

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

func Test_getSignedAttestationsByPubKey(t *testing.T) {
	pubKeys := [][48]byte{
		{1},
	}
	ctx := context.Background()
	validatorDB := dbtest.SetupDB(t, pubKeys)

	// No attestation history stored should return empty.
	signedAttestations, err := getSignedAttestationsByPubKey(ctx, validatorDB, pubKeys[0])
	require.NoError(t, err)
	assert.Equal(t, 0, len(signedAttestations))

	// Storing an attestation history but not the lowest signed target epoch
	// should then return empty.
	history := kv.NewAttestationHistoryArray(0)
	require.NoError(t, validatorDB.SaveAttestationHistoryForPubKeyV2(
		ctx, pubKeys[0], history,
	))
	signedAttestations, err = getSignedAttestationsByPubKey(ctx, validatorDB, pubKeys[0])
	require.NoError(t, err)
	assert.Equal(t, 0, len(signedAttestations))

	// We write a real attesting history to disk for the public key.
	lowestSourceEpoch := uint64(0)
	lowestTargetEpoch := uint64(4)
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, pubKeys[0], lowestSourceEpoch))
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, pubKeys[0], lowestTargetEpoch))

	latestWrittenEpoch := uint64(6)
	newHistory, err := history.SetLatestEpochWritten(ctx, latestWrittenEpoch)
	require.NoError(t, err)
	history = newHistory
	for i := lowestTargetEpoch; i <= latestWrittenEpoch; i++ {
		newHistory, err := history.SetTargetData(ctx, i, &kv.HistoryData{
			Source:      lowestSourceEpoch,
			SigningRoot: []byte(fmt.Sprintf("%d", i)),
		})
		require.NoError(t, err)
		history = newHistory
	}
	require.NoError(t, validatorDB.SaveAttestationHistoryForPubKeyV2(
		ctx, pubKeys[0], history,
	))

	// We then retrieve the signed attestations and expect a correct result.
	signedAttestations, err = getSignedAttestationsByPubKey(ctx, validatorDB, pubKeys[0])
	require.NoError(t, err)
	assert.Equal(t, (latestWrittenEpoch-lowestTargetEpoch)+1, uint64(len(signedAttestations)))

	wanted := []*SignedAttestation{
		{
			SourceEpoch: "0",
			TargetEpoch: "4",
			SigningRoot: "0x3400000000000000000000000000000000000000000000000000000000000000",
		},
		{
			SourceEpoch: "0",
			TargetEpoch: "5",
			SigningRoot: "0x3500000000000000000000000000000000000000000000000000000000000000",
		},
		{
			SourceEpoch: "0",
			TargetEpoch: "6",
			SigningRoot: "0x3600000000000000000000000000000000000000000000000000000000000000",
		},
	}
	assert.DeepEqual(t, wanted, signedAttestations)
}

func Test_getSignedBlocksByPubKey(t *testing.T) {
	pubKeys := [][48]byte{
		{1},
	}
	ctx := context.Background()
	validatorDB := dbtest.SetupDB(t, pubKeys)

	// No highest and/or lowest signed blocks will return empty.
	signedBlocks, err := getSignedBlocksByPubKey(ctx, validatorDB, pubKeys[0])
	require.NoError(t, err)
	assert.Equal(t, 0, len(signedBlocks))

	// We mark slot 1 as proposed.
	dummyRoot1 := [32]byte{1}
	err = validatorDB.SaveProposalHistoryForSlot(ctx, pubKeys[0], 1, dummyRoot1[:])
	require.NoError(t, err)

	// We mark slot 3 as proposed but with empty signing root.
	err = validatorDB.SaveProposalHistoryForSlot(ctx, pubKeys[0], 3, nil)
	require.NoError(t, err)

	// We mark slot 5 as proposed.
	dummyRoot2 := [32]byte{2}
	err = validatorDB.SaveProposalHistoryForSlot(ctx, pubKeys[0], 5, dummyRoot2[:])
	require.NoError(t, err)

	// We expect a valid proposal history containing slot 1 and slot 5 only
	// when we attempt to retrieve it from disk.
	signedBlocks, err = getSignedBlocksByPubKey(ctx, validatorDB, pubKeys[0])
	require.NoError(t, err)
	wanted := []*SignedBlock{
		{
			Slot:        "1",
			SigningRoot: fmt.Sprintf("%#x", dummyRoot1),
		},
		{
			Slot:        "3",
			SigningRoot: "0x0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			Slot:        "5",
			SigningRoot: fmt.Sprintf("%#x", dummyRoot2),
		},
	}
	for i, blk := range wanted {
		assert.DeepEqual(t, blk, signedBlocks[i])
	}
}
