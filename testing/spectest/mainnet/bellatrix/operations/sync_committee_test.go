package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/spectest/shared/bellatrix/operations"
)

func TestMainnet_Bellatrix_Operations_SyncCommittee(t *testing.T) {
	operations.RunSyncCommitteeTest(t, "mainnet")
}
