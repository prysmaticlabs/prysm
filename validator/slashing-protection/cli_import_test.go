package slashingprotection

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	"github.com/prysmaticlabs/prysm/validator/flags"
	spTest "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/testing"
	"github.com/urfave/cli/v2"
)

func TestImportSlashingProtectionCLI(t *testing.T) {
	protectionDir := filepath.Join(t.TempDir(), "protection")
	require.NoError(t, os.MkdirAll(protectionDir, params.BeaconIoConfig().ReadWriteExecutePermissions))

	ctx := context.Background()
	numValidators := 10
	publicKeys := spTest.CreateRandomPubKeys(t, numValidators)
	validatorDB, err := kv.NewKVStore(protectionDir, publicKeys)
	if err != nil {
		t.Fatalf("Failed to instantiate DB: %v", err)
	}
	t.Cleanup(func() {
		if err := validatorDB.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}
		if err := validatorDB.ClearDB(); err != nil {
			t.Fatalf("Failed to clear database: %v", err)
		}
	})

	// First we setup some mock attesting and proposal histories and create a mock
	// standard slashing protection format JSON struct.
	attestingHistory, proposalHistory := spTest.MockAttestingAndProposalHistories(t, numValidators)
	standardProtectionFormat := spTest.MockSlashingProtectionJSON(t, publicKeys, attestingHistory, proposalHistory)

	// We encode the standard slashing protection struct into a JSON format.
	blob, err := json.Marshal(standardProtectionFormat)
	require.NoError(t, err)

	// Write a password for the accounts we wish to backup to a file.
	protectionFilePath := filepath.Join(protectionDir, "protection.json")
	err = ioutil.WriteFile(
		protectionFilePath,
		blob,
		params.BeaconIoConfig().ReadWritePermissions,
	)
	require.NoError(t, err)

	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(cmd.DataDirFlag.Name, protectionDir, "")
	set.String(flags.SlashingProtectionJSONFileFlag.Name, protectionFilePath, "")
	require.NoError(t, set.Set(cmd.DataDirFlag.Name, protectionDir))
	require.NoError(t, set.Set(flags.SlashingProtectionJSONFileFlag.Name, protectionFilePath))
	cliCtx := cli.NewContext(&app, set, nil)

	require.NoError(t, ImportSlashingProtectionCLI(cliCtx, validatorDB))

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
			receivedProposalSigningRoot, _, err := validatorDB.ProposalHistoryForSlot(ctx, publicKeys[i], proposal.Slot)
			require.NoError(t, err)
			require.DeepEqual(
				t,
				receivedProposalSigningRoot[:],
				proposal.SigningRoot,
				"Imported proposals are different then the generated ones",
			)
		}
	}
}
