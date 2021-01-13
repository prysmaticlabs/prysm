package spectest

import (
	"testing"
)

func TestVoluntaryExitMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runVoluntaryExitTest(t, "mainnet")
}
