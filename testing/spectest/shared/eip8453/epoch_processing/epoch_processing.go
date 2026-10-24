package epoch_processing

import (
	"context"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/eip8453"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
)

// RunBuilderPendingPaymentsTest executes "epoch_processing/builder_pending_payments" tests.
func RunBuilderPendingPaymentsTest(t *testing.T, config string) {
	runStateTest(t, config, "epoch_processing", "builder_pending_payments", eip8453.ProcessBuilderPendingPayments)
}

// RunPendingDepositsTest executes "epoch_processing/pending_deposits" tests.
func RunPendingDepositsTest(t *testing.T, config string) {
	runStateTest(t, config, "epoch_processing", "pending_deposits", func(s state.BeaconState) error {
		return eip8453.ProcessPendingDeposits(context.Background(), s)
	})
}

// RunInactivityUpdatesTest executes "round_processing/inactivity_updates" tests.
func RunInactivityUpdatesTest(t *testing.T, config string) {
	runStateTest(t, config, "round_processing", "inactivity_updates", func(s state.BeaconState) error {
		return eip8453.ProcessInactivityUpdates(context.Background(), s)
	})
}

// RunRewardsTest executes "round_processing/rewards" tests.
func RunRewardsTest(t *testing.T, config string) {
	runStateTest(t, config, "round_processing", "rewards", func(s state.BeaconState) error {
		return eip8453.ProcessRewards(context.Background(), s)
	})
}

// RunParticipationFlagUpdatesTest executes "round_processing/participation_flag_updates" tests.
func RunParticipationFlagUpdatesTest(t *testing.T, config string) {
	runStateTest(t, config, "round_processing", "participation_flag_updates", eip8453.ProcessParticipationFlagUpdates)
}
