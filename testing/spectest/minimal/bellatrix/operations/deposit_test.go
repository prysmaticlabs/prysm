package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/bellatrix/operations"
)

func TestMinimal_Merge_Operations_Deposit(t *testing.T) {
	operations.RunDepositTest(t, "minimal")
}
