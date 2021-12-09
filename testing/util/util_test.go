package util

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/params"
)

func TestMain(m *testing.M) {
	prevConfig := params.BeaconConfig().Copy()
	defer params.OverrideBeaconConfig(prevConfig)
	cfg := params.MinimalSpecConfig()
	params.OverrideBeaconConfig(cfg)

	m.Run()
}
