package slashingprotection

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_ImportInterchangeData_OK(t *testing.T) {
	ctx := context.Background()
	numValidators := 10
	publicKeys := createRandomPubKeys(t, numValidators)
	validatorDB := dbtest.SetupDB(t, publicKeys)

	// First we setup some mock attesting and proposal histories and create a mock
	// standard slashing protection format JSON struct.
	attestingHistory, proposalHistory := mockAttestingAndProposalHistories(t, numValidators)
	standardProtectionFormat := mockSlashingProtectionJSON(t, publicKeys, attestingHistory, proposalHistory)

	// We encode the standard slashing protection struct into a JSON format.
	blob, err := json.Marshal(standardProtectionFormat)
	require.NoError(t, err)
	buf := bytes.NewBuffer(blob)

	// Next, we attempt to import it into our validator database.
	err = ImportStandardProtectionJSON(ctx, validatorDB, buf)
	require.NoError(t, err)

	// Next, we attempt to retrieve the attesting and proposals histories from our database and
	// verify those indeed match the originally generated mock histories.
	receivedAttestingHistory, err := validatorDB.AttestationHistoryForPubKeysV2(ctx, publicKeys)
	require.NoError(t, err)
	for i := 0; i < len(publicKeys); i++ {
		require.DeepEqual(
			t,
			attestingHistory[i],
			receivedAttestingHistory[publicKeys[i]],
			"We should have stored any attesting history",
		)
		proposals := proposalHistory[i].Proposals
		for _, proposal := range proposals {
			receivedProposalSigningRoot, err := validatorDB.ProposalHistoryForSlot(ctx, publicKeys[i][:], proposal.Slot)
			require.NoError(t, err)
			require.DeepEqual(
				t,
				receivedProposalSigningRoot,
				proposal.SigningRoot,
				"Imported proposals are different then the generated ones",
			)
		}
	}
}
