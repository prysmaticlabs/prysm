package forkchoice

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/common/forkchoice"
)

func TestMainnet_Altair_Forkchoice(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		EnableDefensivePull: false,
		DisablePullTips:     true,
	})
	defer resetCfg()
	forkchoice.Run(t, "mainnet", version.Altair)
}
