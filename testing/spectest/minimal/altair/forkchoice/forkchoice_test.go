package forkchoice

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/common/forkchoice"
)

func TestMinimal_Altair_Forkchoice(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		DisablePullTips:     true,
		EnableDefensivePull: false,
	})
	defer resetCfg()
	forkchoice.Run(t, "minimal", version.Altair)
}
