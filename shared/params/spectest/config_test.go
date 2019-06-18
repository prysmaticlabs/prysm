package spectest

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestConfig(t *testing.T) {
	SetConfig("minimal")
	if params.BeaconConfig().SlotsPerEpoch != 8 {
		t.Errorf("Expected minimal config to be set, but got %d slots per epoch", params.BeaconConfig().SlotsPerEpoch)
	}

	SetConfig("mainnet")
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("Expected mainnet config to be set, but got %d slots per epoch", params.BeaconConfig().SlotsPerEpoch)
	}
}
