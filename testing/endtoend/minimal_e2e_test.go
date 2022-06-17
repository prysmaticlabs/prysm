package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	e2eMinimal(t).run()
}

func TestEndToEnd_MinimalConfig_Web3Signer(t *testing.T) {
	e2eMinimal(t, types.WithRemoteSigner()).run()
}

func TestEndToEnd_MinimalConfig_CheckpointSync(t *testing.T) {
	e2eMinimal(t, types.WithCheckpointSync()).run()
}
