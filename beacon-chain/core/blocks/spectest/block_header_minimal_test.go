package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestBlockHeaderMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(blkHeaderPrefix + "block_header_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runBlockHeaderTest(t, filepath)
}
