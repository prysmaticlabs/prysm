package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/deneb/operations"
)

func TestMinimal_Deneb_Operations_Deposit(t *testing.T) {
	operations.RunDepositTest(t, "minimal")
}
