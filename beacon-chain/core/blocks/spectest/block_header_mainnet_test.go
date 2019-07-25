package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestBlockHeaderMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(blkHeaderPrefix + "block_header_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runBlockHeaderTest(t, filepath)
}
