package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestSlashingsMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(slashingsPrefix + "slashings_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runSlashingsTests(t, filepath)
}
