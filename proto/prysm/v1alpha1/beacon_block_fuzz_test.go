package eth_test

import (
	"testing"

	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

func TestCopyBeaconBlockFields_Fuzz(t *testing.T) {
	fuzzCopies(t, &eth.Eth1Data{})
	fuzzCopies(t, &eth.ProposerSlashing{})
	fuzzCopies(t, &eth.SignedBeaconBlockHeader{})
	fuzzCopies(t, &eth.BeaconBlockHeader{})
	fuzzCopies(t, &eth.Deposit{})
	fuzzCopies(t, &eth.Deposit_Data{})
	fuzzCopies(t, &eth.SignedVoluntaryExit{})
	fuzzCopies(t, &eth.VoluntaryExit{})
	fuzzCopies(t, &eth.SyncAggregate{})
	fuzzCopies(t, &eth.SignedBLSToExecutionChange{})
	fuzzCopies(t, &eth.BLSToExecutionChange{})
	fuzzCopies(t, &eth.HistoricalSummary{})
	fuzzCopies(t, &eth.PendingBalanceDeposit{})
}
