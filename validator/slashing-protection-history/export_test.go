package history

import (
	"context"
	"fmt"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection-history/format"
)

func TestExportStandardProtectionJSON_EmptyGenesisRoot(t *testing.T) {
	ctx := context.Background()
	pubKeys := [][fieldparams.BLSPubkeyLength]byte{
		{1},
	}
	validatorDB := dbtest.SetupDB(t, pubKeys)
	_, err := ExportStandardProtectionJSON(ctx, validatorDB)
	require.ErrorContains(t, "genesis validators root is empty", err)
	genesisValidatorsRoot := [32]byte{1}
	err = validatorDB.SaveGenesisValidatorsRoot(ctx, genesisValidatorsRoot[:])
	require.NoError(t, err)
	_, err = ExportStandardProtectionJSON(ctx, validatorDB)
	require.NoError(t, err)
}

func Test_getSignedAttestationsByPubKey(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		pubKeys := [][fieldparams.BLSPubkeyLength]byte{
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
	})
	t.Run("old_schema_bug_edge_case_genesis", func(t *testing.T) {
		pubKeys := [][fieldparams.BLSPubkeyLength]byte{
			{1},
		}
		ctx := context.Background()
		validatorDB := dbtest.SetupDB(t, pubKeys)

		// No attestation history stored should return empty.
		signedAttestations, err := signedAttestationsByPubKey(ctx, validatorDB, pubKeys[0])
		require.NoError(t, err)
		assert.Equal(t, 0, len(signedAttestations))

		// We write a real attesting history to disk for the public key with
		// source epoch 0 and target epoch 1000.
		lowestSourceEpoch := types.Epoch(0)
		lowestTargetEpoch := types.Epoch(1000)

		// Next up, we simulate a DB affected by the bug where the next entry
		// has a target epoch less than the previous one.
		require.NoError(t, validatorDB.SaveAttestationForPubKey(ctx, pubKeys[0], [32]byte{4}, createAttestation(
			lowestSourceEpoch,
			lowestTargetEpoch,
		)))
		require.NoError(t, validatorDB.SaveAttestationForPubKey(ctx, pubKeys[0], [32]byte{5}, createAttestation(
			1,
			2,
		)))

		// We then retrieve the signed attestations and expect to have
		// skipped the 0th, corrupted entry.
		signedAttestations, err = signedAttestationsByPubKey(ctx, validatorDB, pubKeys[0])
		require.NoError(t, err)

		wanted := []*format.SignedAttestation{
			{
				SourceEpoch: "1",
				TargetEpoch: "2",
				SigningRoot: "0x0500000000000000000000000000000000000000000000000000000000000000",
			},
		}
		assert.DeepEqual(t, wanted, signedAttestations)
	})
	t.Run("old_schema_bug_edge_case_not_genesis", func(t *testing.T) {
		pubKeys := [][fieldparams.BLSPubkeyLength]byte{
			{1},
		}
		ctx := context.Background()
		validatorDB := dbtest.SetupDB(t, pubKeys)

		// No attestation history stored should return empty.
		signedAttestations, err := signedAttestationsByPubKey(ctx, validatorDB, pubKeys[0])
		require.NoError(t, err)
		assert.Equal(t, 0, len(signedAttestations))

		// We write a real attesting history to disk for the public key with
		// source epoch 1 and target epoch 1000.
		lowestSourceEpoch := types.Epoch(1)
		lowestTargetEpoch := types.Epoch(1000)

		// Next up, we simulate a DB affected by the bug where the next entry
		// has a target epoch less than the previous one.
		require.NoError(t, validatorDB.SaveAttestationForPubKey(ctx, pubKeys[0], [32]byte{4}, createAttestation(
			lowestSourceEpoch,
			lowestTargetEpoch,
		)))
		require.NoError(t, validatorDB.SaveAttestationForPubKey(ctx, pubKeys[0], [32]byte{5}, createAttestation(
			1,
			2,
		)))

		// We then retrieve the signed attestations and do not expect changes
		// as the bug only manifests in the genesis epoch.
		signedAttestations, err = signedAttestationsByPubKey(ctx, validatorDB, pubKeys[0])
		require.NoError(t, err)

		wanted := []*format.SignedAttestation{
			{
				SourceEpoch: "1",
				TargetEpoch: "1000",
				SigningRoot: "0x0400000000000000000000000000000000000000000000000000000000000000",
			},
			{
				SourceEpoch: "1",
				TargetEpoch: "2",
				SigningRoot: "0x0500000000000000000000000000000000000000000000000000000000000000",
			},
		}
		assert.DeepEqual(t, wanted, signedAttestations)
	})
}

func Test_getSignedBlocksByPubKey(t *testing.T) {
	pubKeys := [][fieldparams.BLSPubkeyLength]byte{
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
