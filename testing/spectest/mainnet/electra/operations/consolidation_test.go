package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/operations"
)

func TestMainnet_Electra_Operations_Consolidation(t *testing.T) {
	t.Skip("These tests were temporarily deleted in v1.5.0-alpha.2. See https://github.com/ethereum/consensus-specs/pull/3736")
	operations.RunConsolidationTest(t, "mainnet")
}
