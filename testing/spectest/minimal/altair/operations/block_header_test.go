package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/operations"
)

func TestMinimal_Altair_Operations_BlockHeader(t *testing.T) {
	operations.RunBlockHeaderTest(t, "minimal")
}
