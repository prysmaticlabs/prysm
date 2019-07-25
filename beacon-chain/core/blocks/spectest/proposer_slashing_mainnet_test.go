package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestProposerSlashingMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(proposerSlashingPrefix + "proposer_slashing_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runProposerSlashingTest(t, filepath)
}
