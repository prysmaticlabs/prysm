package fork_transition

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/fork"
)

func TestMinimal_Electra_Transition(t *testing.T) {
	t.Skip("TODO: Electra")
	fork.RunForkTransitionTest(t, "minimal")
}
