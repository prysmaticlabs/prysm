package endtoend

import (
	"testing"
)

func TestEndToEnd_MainnetConfig_MultiClient(t *testing.T) {
	t.Skip()
	e2eMainnet(t, false /*usePrysmSh*/, true /*useMultiClient*/)
}
