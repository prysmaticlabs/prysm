package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestJustificationAndFinalizationMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(justificationAndFinalizationPrefix + "justification_and_finalization_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runJustificationAndFinalizationTests(t, filepath)
}
