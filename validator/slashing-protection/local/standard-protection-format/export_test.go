package interchangeformat

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format/format"
)

func Test_getSignedAttestationsByPubKey(t *testing.T) {
	pubKeys := [][48]byte{
		{1},
	}
	ctx := context.Background()
	validatorDB := dbtest.SetupDB(t, pubKeys)

	// No attestation history stored should return empty.
	signedAttestations, err := signedAttestationsByPubKey(ctx, validatorDB, pubKeys[0])
	require.NoError(t, err)
	assert.Equal(t, 0, len(signedAttestations))

	// We write a real attesting history to disk for the public key.
	lowestSourceEpoch := types.Epoch(0)
	lowestTargetEpoch := types.Epoch(4)

	require.NoError(t, validatorDB.SaveAttestationForPubKey(ctx, pubKeys[0], [32]byte{4}, createAttestation(
		lowestSourceEpoch,
		lowestTargetEpoch,
	)))
	require.NoError(t, validatorDB.SaveAttestationForPubKey(ctx, pubKeys[0], [32]byte{5}, createAttestation(
		lowestSourceEpoch,
		lowestTargetEpoch+1,
	)))

	// We then retrieve the signed attestations and expect a correct result.
	signedAttestations, err = signedAttestationsByPubKey(ctx, validatorDB, pubKeys[0])
	require.NoError(t, err)

	wanted := []*format.SignedAttestation{
		{
			SourceEpoch: "0",
			TargetEpoch: "4",
			SigningRoot: "0x0400000000000000000000000000000000000000000000000000000000000000",
		},
		{
			SourceEpoch: "0",
			TargetEpoch: "5",
			SigningRoot: "0x0500000000000000000000000000000000000000000000000000000000000000",
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
	signedBlocks, err := signedBlocksByPubKey(ctx, validatorDB, pubKeys[0])
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
	signedBlocks, err = signedBlocksByPubKey(ctx, validatorDB, pubKeys[0])
	require.NoError(t, err)
	wanted := []*format.SignedBlock{
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
