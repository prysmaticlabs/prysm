package finality

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/deneb/finality"
)

func TestMainnet_Deneb_Finality(t *testing.T) {
	finality.RunFinalityTest(t, "mainnet")
}
