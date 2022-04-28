package endtoend

import (
	"testing"
)

func TestEndToEnd_MainnetConfig_MultiClient(t *testing.T) {
	e2eMainnet(t, false /*usePrysmSh*/, true /*useMultiClient*/)
}
