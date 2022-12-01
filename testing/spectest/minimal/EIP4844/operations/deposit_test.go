package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/operations"
)

func TestMinimal_EIP4844_Operations_Deposit(t *testing.T) {
	operations.RunDepositTest(t, "minimal")
}
