package forkchoice

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/forkchoice"
)

func TestMainnet_Altair_Forkchoice(t *testing.T) {
	forkchoice.RunTest(t, "mainnet")
}
