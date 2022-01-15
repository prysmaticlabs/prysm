package forkchoice

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/forkchoice"
)

func TestMinimal_Altair_Forkchoice(t *testing.T) {
	forkchoice.RunTest(t, "minimal")
}
