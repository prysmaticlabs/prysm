package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/altair/operations"
)

func TestMinimal_Altair_Operations_SyncCommittee(t *testing.T) {
	operations.RunProposerSlashingTest(t, "minimal")
}
