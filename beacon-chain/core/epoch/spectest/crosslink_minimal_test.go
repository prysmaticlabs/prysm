package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestCrosslinksProcessingMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(crosslinkPrefix + "crosslinks_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runCrosslinkProcessingTests(t, filepath)
}
