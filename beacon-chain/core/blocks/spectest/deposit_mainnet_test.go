package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestDepositMainnetYaml(t *testing.T) {
	filepath, err := bazel.Runfile(depositPrefix + "deposit_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runDepositTest(t, filepath)
}
