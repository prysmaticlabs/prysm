package forkchoice

import (
	"testing"

	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/spectest/shared/common/forkchoice"
)

func TestMinimal_Altair_Forkchoice(t *testing.T) {
	forkchoice.Run(t, "minimal", version.Phase0)
}

func TestMinimal_Altair_Forkchoice_DoublyLinkTre(t *testing.T) {
	resetCfg := features.InitWithReset(&features.Flags{
		EnableForkChoiceDoublyLinkedTree: true,
	})
	defer resetCfg()
	forkchoice.Run(t, "minimal", version.Phase0)
}
