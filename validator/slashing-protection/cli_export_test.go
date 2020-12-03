package slashingprotection

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	dbTest "github.com/prysmaticlabs/prysm/validator/db/testing"
	"github.com/prysmaticlabs/prysm/validator/flags"
	protectionFormat "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/standard-protection-format"
	mocks "github.com/prysmaticlabs/prysm/validator/testing"
	"github.com/urfave/cli/v2"
)

func setupCliCtx(
	tb testing.TB,
	dbPath string,
	outputDir string,
) *cli.Context {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	set.String(cmd.DataDirFlag.Name, dbPath, "")
	set.String(flags.SlashingProtectionExportDirFlag.Name, outputDir, "")
	assert.NoError(tb, set.Set(cmd.DataDirFlag.Name, dbPath))
	assert.NoError(tb, set.Set(flags.SlashingProtectionExportDirFlag.Name, outputDir))
	return cli.NewContext(&app, set, nil)
}

func TestExportSlashingProtectionCli(t *testing.T) {
	ctx := context.Background()
	numValidators := 10

	// Create some mock slashing protection history. and JSON file
	pubKeys, err := mocks.CreateRandomPubKeys(numValidators)
	require.NoError(t, err)
	attestingHistory, proposalHistory, err := mocks.MockAttestingAndProposalHistories(numValidators)
	require.NoError(t, err)
	mockJSON, err := mocks.MockSlashingProtectionJSON(pubKeys, attestingHistory, proposalHistory)
	require.NoError(t, err)

	// We JSON encode the protection file and import it into our database.
	encoded, err := json.Marshal(mockJSON)
	require.NoError(t, err)
	buf := bytes.NewBuffer(encoded)

	validatorDB := dbTest.SetupDB(t, pubKeys)
	err = protectionFormat.ImportStandardProtectionJSON(ctx, validatorDB, buf)
	require.NoError(t, err)
	require.NoError(t, validatorDB.Close())

	// We export our slashing protection history by creating a CLI context
	// with the required values, such as the database datadir and output directory.
	dbPath := validatorDB.DatabasePath()
	outputPath := filepath.Join(os.TempDir(), "slashing-exports")
	cliCtx := setupCliCtx(t, dbPath, outputPath)
	err = ExportSlashingProtectionJSONCli(cliCtx)
	require.NoError(t, err)

	// Attempt to read the exported file from the output directory.
	enc, err := fileutil.ReadFileAsBytes(filepath.Join(outputPath, jsonExportFileName))
	require.NoError(t, err)

	receivedJSON := &protectionFormat.EIPSlashingProtectionFormat{}
	err = json.Unmarshal(enc, receivedJSON)
	require.NoError(t, err)

	// We verify the parsed JSON file matches. Given there is no guarantee of order,
	// we will have to carefully compare and sort values as needed.
	//
	// First, we compare basic data such as the Metadata value in the JSON file.
	require.DeepEqual(t, mockJSON.Metadata, receivedJSON.Metadata)
}
