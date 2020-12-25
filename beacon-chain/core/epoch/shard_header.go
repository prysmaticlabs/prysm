package epoch

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	eth "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ProcessPendingHeaders for beacon chain.
func ProcessPendingHeaders(state *stateTrie.BeaconState) (*stateTrie.BeaconState, error) {
	for slot := uint64(0); slot < params.BeaconConfig().SlotsPerEpoch; slot++ {
		for shard := uint64(0); shard < helpers.ActiveShardCount(); shard++ {
			var candidates []*eth.PendingShardHeader
			for _, header := range state.PreviousEpochPendingShardHeaders() {
				if header.Slot == slot && header.Shard == shard && !header.Confirmed {
					candidates = append(candidates, header)
				}
			}
			fullCommittee, err := helpers.BeaconCommitteeFromState(state, slot, shard)
			if err != nil {
				return nil, err
			}
			var bestIndex int
			var bestBalance uint64
			for i, candidate := range candidates {
				attestedCommittee := attestationutil.AttestingIndices(candidate.Votes, fullCommittee)
				attestedBalance := helpers.TotalBalance(state, attestedCommittee)
				if attestedBalance > bestBalance {
					bestBalance = attestedBalance
					bestIndex = i
				}
			}
			candidates[bestIndex].Confirmed = true

			// Replace candidate
		}
	}

	// Update most recent commitments

	return state, nil
}

// ChargeConfirmedHeaderFees for beacon chain.
func ChargeConfirmedHeaderFees(state *stateTrie.BeaconState) (*stateTrie.BeaconState, error) {
	// newGasPrice := state.ShardGasPrice()
	// adjustmentQuotient := helpers.ActiveShardCount() * params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().GaspriceAdjustmentCoefficient
	prevHeaders := state.PreviousEpochPendingShardHeaders()
	for slot := uint64(0); slot < params.BeaconConfig().SlotsPerEpoch; slot++ {
		for shard := uint64(0); shard < helpers.ActiveShardCount(); shard++ {
			var h *eth.PendingShardHeader
			for _, header := range prevHeaders {
				if header.Slot == slot && header.Shard == shard && !header.Confirmed {
					h = header
					break
				}
			}
			if h != nil {
				// Charge EIP fee
			}
		}
	}
	// Set new gas price
	return state, nil
}

// ResetPendingHeaders for beacon chain.
