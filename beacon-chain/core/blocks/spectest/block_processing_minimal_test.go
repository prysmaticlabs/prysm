package spectest

import (
	"testing"
)

func TestBlockProcessingMinimalYaml(t *testing.T) {
	t.Skip("This test suite requires --define ssz=minimal to be provided and there isn't a great way to do that without breaking //... See https://github.com/prysmaticlabs/prysm/issues/3066")

	runBlockProcessingTest(t, "sanity_blocks_minimal.yaml")
}
