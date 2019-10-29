package spectest

import (
	"testing"
)

func TestJustificationAndFinalizationMainnet(t *testing.T) {
	t.Skip("Disabled until v0.9.0 (#3865) completes")
	runJustificationAndFinalizationTests(t, "mainnet")
}
