package endtoend

import (
	"testing"
)

func TestEndToEnd_MinimalConfig(t *testing.T) {
	e2eMinimal(t, false, 3).run()
}

func TestEndToEnd_MinimalConfig_Web3Signer(t *testing.T) {
	e2eMinimal(t, true, 0).run()
}
