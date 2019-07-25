package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestSlashingsMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(slashingsPrefix + "slashings_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runSlashingsTests(t, filepath)
}
