package spectest

import (
	"testing"
)

func TestBlockProcessingMainnetYaml(t *testing.T) {
	t.Skip("Skip until 3960 merges")
	runBlockProcessingTest(t, "mainnet")
}
