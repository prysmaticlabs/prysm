package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	e2eMinimal(t, types.WithCheckpointSync()).run()
}
