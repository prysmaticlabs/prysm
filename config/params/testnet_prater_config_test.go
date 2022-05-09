package params_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestPraterConfigMatchesUpstreamYaml(t *testing.T) {
	presetFPs := presetsFilePath(t, "mainnet")
	for _, fp := range presetFPs {
		require.NoError(t, params.LoadChainConfigFile(fp, nil))
	}
	configFP := testnetConfigFilePath(t, "prater")
	require.NoError(t, params.LoadChainConfigFile(configFP, nil))
	fields := fieldsFromYamls(t, append(presetFPs, configFP))
	assertYamlFieldsMatch(t, "prater", fields, params.BeaconConfig(), params.PraterConfig())
}
