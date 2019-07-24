package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestJustificationAndFinalizationMinimal(t *testing.T) {
	t.Skip("This test suite requires --define ssz=minimal to be provided and there isn't a great way to do that without breaking //... See https://github.com/prysmaticlabs/prysm/issues/3066")
	filepath, err := bazel.Runfile(justificationAndFinalizationPrefix + "justification_and_finalization_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runJustificationAndFinalizationTests(t, filepath)
}
