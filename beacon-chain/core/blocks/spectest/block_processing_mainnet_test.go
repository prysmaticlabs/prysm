package spectest

import (
	"testing"
)

func TestBlockProcessingMainnetYaml(t *testing.T) {
	runBlockProcessingTest(t, "mainnet")
}
