package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

func TestEndToEnd_MinimalConfigX(t *testing.T) {
	e2eMinimal(t, types.WithCheckpointSync()).run()
}
