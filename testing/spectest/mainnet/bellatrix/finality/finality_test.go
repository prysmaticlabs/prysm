package finality

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/bellatrix/finality"
)

func TestMainnet_Merge_Finality(t *testing.T) {
	finality.RunFinalityTest(t, "mainnet")
}
