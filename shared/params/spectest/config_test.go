package spectest

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestConfig(t *testing.T) {
	if err := SetConfig(t, "minimal"); err != nil {
		t.Fatal(err)
	}
	if params.BeaconConfig().SlotsPerEpoch != 8 {
		t.Errorf("Expected minimal config to be set, but got %d slots per epoch", params.BeaconConfig().SlotsPerEpoch)
	}

	if err := SetConfig(t, "mainnet"); err != nil {
		t.Fatal(err)
	}
	if params.BeaconConfig().SlotsPerEpoch != 32 {
		t.Errorf("Expected mainnet config to be set, but got %d slots per epoch", params.BeaconConfig().SlotsPerEpoch)
	}
}
