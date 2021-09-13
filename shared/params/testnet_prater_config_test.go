package params

import (
	"testing"
)

func TestPraterConfigMatchesUpstreamYaml(t *testing.T) {
	configFP := testnetConfigFilePath(t, "prater")
	LoadChainConfigFile(configFP)
	fields := fieldsFromYaml(t, configFP)
	assertYamlFieldsMatch(t, "prater", fields, BeaconConfig(), PraterConfig())
}
