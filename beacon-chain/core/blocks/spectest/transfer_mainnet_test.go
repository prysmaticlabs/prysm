package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestTransferMainnet(t *testing.T) {
	t.Skip("Transfer tests are disabled. See https://github.com/ethereum/eth2.0-specs/pull/1238#issuecomment-507054595")
	filepath, err := bazel.Runfile(transferPrefix + "transfer_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runTransferTest(t, filepath)
}
