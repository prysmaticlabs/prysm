package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestDepositMinimalYaml(t *testing.T) {
	filepath, err := bazel.Runfile(depositPrefix + "deposit_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runDepositTest(t, filepath)
}
