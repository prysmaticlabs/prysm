package genesis

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/testing/spectest/shared/phase0/genesis"
)

func TestMinimal_Phase0_Genesis_Initialization(t *testing.T) {
	genesis.RunInitializationTest(t, "minimal")
}
