package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
)

func TestCrosslinksProcessingMainnet(t *testing.T) {
	helpers.ClearAllCaches()
	filepath, err := bazel.Runfile(crosslinkPrefix + "crosslinks_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runCrosslinkProcessingTests(t, filepath)
}
