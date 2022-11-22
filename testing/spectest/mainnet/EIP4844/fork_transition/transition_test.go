package fork_transition

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/fork"
)

func TestMainnet_EIP4844_Transition(t *testing.T) {
	fork.RunForkTransitionTest(t, "mainnet")
}
