package spectest

import (
	"testing"
)

func TestBlockProcessingMainnetYaml(t *testing.T) {
	runBlockProcessingTest(t, "sanity_blocks_mainnet.yaml")
}
