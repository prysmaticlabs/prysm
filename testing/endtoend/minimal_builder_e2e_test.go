package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/testing/endtoend/types"
)

func TestEndToEnd_MinimalConfig_WithBuilder(t *testing.T) {
	r := e2eMinimal(t, version.Phase0, types.WithCheckpointSync(), types.WithBuilder())
	r.run()
}

func TestEndToEnd_MinimalConfig_WithBuilder_ValidatorRESTApi(t *testing.T) {
	r := e2eMinimal(t, version.Phase0, types.WithCheckpointSync(), types.WithBuilder(), types.WithValidatorRESTApi())
	r.run()
}
