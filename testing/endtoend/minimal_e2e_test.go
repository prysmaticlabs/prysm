package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	r := e2eMinimal(t, types.StartAtBellatrix(params.E2ETestConfig()), types.WithCheckpointSync())
	r.run()
}
