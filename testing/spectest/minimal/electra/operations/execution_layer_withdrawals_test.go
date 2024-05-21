package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/electra/operations"
)

func TestMinimal_Electra_Operations_ExecutionLayerWithdrawalRequest(t *testing.T) {
	operations.RunExecutionLayerWithdrawalRequestTest(t, "minimal")
}
