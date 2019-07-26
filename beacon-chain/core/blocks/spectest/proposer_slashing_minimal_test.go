package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestProposerSlashingMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(proposerSlashingPrefix + "proposer_slashing_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runProposerSlashingTest(t, filepath)
}
