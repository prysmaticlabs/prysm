package finality

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/finality"
)

func TestMainnet_EIP4844_Finality(t *testing.T) {
	finality.RunFinalityTest(t, "mainnet")
}
