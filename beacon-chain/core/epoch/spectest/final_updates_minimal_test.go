package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestFinalUpdatesMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(finalUpdatesPrefix + "final_updates_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runFinalUpdatesTests(t, filepath)
}
