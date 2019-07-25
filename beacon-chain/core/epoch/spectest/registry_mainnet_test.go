package spectest

import (
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

func TestRegistryProcessingMainnet(t *testing.T) {
	filepath, err := bazel.Runfile(registryUpdatesPrefix + "registry_updates_mainnet.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runRegisteryProcessingTests(t, filepath)
}
