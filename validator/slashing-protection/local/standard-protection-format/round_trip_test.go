package interchangeformat

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

func TestImportExport_RoundTrip(t *testing.T) {
	ctx := context.Background()
	numValidators := 5
	publicKeys := createRandomPubKeys(t, numValidators)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	// First we setup some mock attesting and proposal histories and create a mock
	// standard slashing protection format JSON struct.
	attestingHistory, proposalHistory := mockAttestingAndProposalHistories(t, numValidators)
	wanted := mockSlashingProtectionJSON(t, publicKeys, attestingHistory, proposalHistory)

	// We encode the standard slashing protection struct into a JSON format.
	blob, err := json.Marshal(wanted)
	require.NoError(t, err)
	buf := bytes.NewBuffer(blob)

	// Next, we attempt to import it into our validator database.
	err = ImportStandardProtectionJSON(ctx, validatorDB, buf)
	require.NoError(t, err)

	// Next up, we export our slashing protection database into the EIP standard file.
	// Next, we attempt to import it into our validator database.
	eipStandard, err := ExportStandardProtectionJSON(ctx, validatorDB)
	require.NoError(t, err)

	// TODO(#7813): We have only implemented the export functionality
	// for proposals history at the moment, so we do not check attesting history.
	for i := range wanted.Data {
		wanted.Data[i].SignedAttestations = nil
	}

	// We compare the metadata fields from import to export.
	require.Equal(t, wanted.Metadata, eipStandard.Metadata)

	// The values in the data field of the EIP struct are not guaranteed to be sorted,
	// so we create a map to verify we have the data we expected.
	require.Equal(t, len(wanted.Data), len(eipStandard.Data))

	dataByPubKey := make(map[string]*ProtectionData)
	for _, item := range wanted.Data {
		dataByPubKey[item.Pubkey] = item
	}
	for _, item := range eipStandard.Data {
		want, ok := dataByPubKey[item.Pubkey]
		require.Equal(t, true, ok)
		require.DeepEqual(t, want, item)
	}
}
