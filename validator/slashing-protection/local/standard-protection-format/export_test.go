package interchangeformat

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

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
