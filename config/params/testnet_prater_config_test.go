package params_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
)

func TestPraterConfigMatchesUpstreamYaml(t *testing.T) {
	presetFPs := presetsFilePath(t, "mainnet")
	for _, fp := range presetFPs {
		params.LoadChainConfigFile(fp)
	}
	configFP := testnetConfigFilePath(t, "prater")
	params.LoadChainConfigFile(configFP)
	fields := fieldsFromYamls(t, append(presetFPs, configFP))
	assertYamlFieldsMatch(t, "prater", fields, params.BeaconConfig(), params.PraterConfig())
}
