package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestVoluntaryExitMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(exitPrefix + "voluntary_exit_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runVoluntaryExitTest(t, filepath)
}
