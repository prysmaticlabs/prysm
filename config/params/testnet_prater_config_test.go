package params_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
)

func TestPraterConfigMatchesUpstreamYaml(t *testing.T) {
	presetFP := presetsFilePath(t, "mainnet")
	params.LoadChainConfigFile(presetFP)
	configFP := testnetConfigFilePath(t, "prater")
	params.LoadChainConfigFile(configFP)
	fields := fieldsFromYamls(t, []string{configFP, presetFP})
	assertYamlFieldsMatch(t, "prater", fields, params.BeaconConfig(), params.PraterConfig())
}
