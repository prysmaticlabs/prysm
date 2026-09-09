package compliance

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/OffchainLabs/prysm/v7/testing/spectest/shared/common/forkchoice"
)

func TestMinimal_Altair_ForkchoiceCompliance(t *testing.T) {
	forkchoice.RunCompliance(t, "minimal", version.Altair)
}
