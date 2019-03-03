package state

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestReportEpochTransitionMetrics_validatorBalances(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorBalances: []uint64{1, 15},
		ValidatorRegistry: []*pb.Validator{
			{Pubkey: []byte{1}},
			{Pubkey: []byte{2}},
		},
	}

	reportEpochTransitionMetrics(state)
	expectedMetadata := `
	  # HELP state_validator_balances Balances of validators, updated on epoch transition
	  # TYPE state_validator_balances gauge
	`
	expectedValues := `
		state_validator_balances{validator="0x01"} 1
		state_validator_balances{validator="0x02"} 15
	`
	expected := expectedMetadata + expectedValues
	if err := testutil.CollectAndCompare(validatorBalancesGauge, strings.NewReader(expected)); err != nil {
		t.Error(err)
	}
}
