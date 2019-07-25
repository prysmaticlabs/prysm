package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestRegistryProcessingMinimal(t *testing.T) {
	filepath, err := bazel.Runfile(registryUpdatesPrefix + "registry_updates_minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runRegisteryProcessingTests(t, filepath)
}
