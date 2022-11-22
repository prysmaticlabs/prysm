package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/testing/spectest/shared/eip4844/operations"
)

func TestMainnet_EIP4844_Operations_SyncCommittee(t *testing.T) {
	operations.RunSyncCommitteeTest(t, "mainnet")
}
