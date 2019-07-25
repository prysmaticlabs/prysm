package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestFinalUpdatesMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(finalUpdatesPrefix + "final_updates_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runFinalUpdatesTests(t, filepath)
}
