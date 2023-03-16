package forkchoice

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/common/forkchoice"
)

func TestMainnet_Bellatrix_Forkchoice(t *testing.T) {
	forkchoice.Run(t, "mainnet", version.Bellatrix)
}
