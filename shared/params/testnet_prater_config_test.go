package params_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestPraterConfigMatchesUpstreamYaml(t *testing.T) {
	configFP := testnetConfigFilePath(t, "prater")
	params.LoadChainConfigFile(configFP)
	fields := fieldsFromYaml(t, configFP)
	assertYamlFieldsMatch(t, "prater", fields, params.BeaconConfig(), params.PraterConfig())
}
