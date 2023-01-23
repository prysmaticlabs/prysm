package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	r := e2eMinimal(t, version.Phase0, types.WithCheckpointSync())
	r.run()
}
