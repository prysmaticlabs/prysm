package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestAttesterSlashingMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(attesterSlashingPrefix + "attester_slashing_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runAttesterSlashingTest(t, filepath)
}
