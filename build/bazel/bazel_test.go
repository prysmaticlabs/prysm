package bazel_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/build/bazel"
)

func TestBuildWithBazel(t *testing.T) {
	if !bazel.BuiltWithBazel() {
		t.Error("not built with Bazel")
	}
}
