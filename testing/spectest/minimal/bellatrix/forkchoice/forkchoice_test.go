package forkchoice

import (
	"testing"

	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/testing/spectest/shared/common/forkchoice"
)

func TestMinimal_Bellatrix_Forkchoice(t *testing.T) {
	forkchoice.Run(t, "minimal", version.Bellatrix)
}
