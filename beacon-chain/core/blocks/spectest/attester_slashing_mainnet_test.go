package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestAttesterSlashingMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(attesterSlashingPrefix + "attester_slashing_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runAttesterSlashingTest(t, filepath)
}
