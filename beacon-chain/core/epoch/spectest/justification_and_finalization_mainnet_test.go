package spectest

import (
	"testing"
)

func TestJustificationAndFinalizationMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runJustificationAndFinalizationTests(t, "mainnet")
}
