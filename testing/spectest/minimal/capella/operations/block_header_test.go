package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/capella/operations"
)

func TestMinimal_Capella_Operations_BlockHeader(t *testing.T) {
	operations.RunBlockHeaderTest(t, "minimal")
}
