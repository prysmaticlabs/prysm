package spectest

import (
	"testing"
)

func TestBlockProcessingMainnetYaml(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runBlockProcessingTest(t, "mainnet")
}
