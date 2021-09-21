package merge

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// UpgradeToMerge updates input state to return the version Merge state.
//
// Spec code:
// def upgrade_to_merge(pre: altair.BeaconState) -> BeaconState:
//    epoch = altair.get_current_epoch(pre)
//    post = BeaconState(
//        # Versioning
//        genesis_time=pre.genesis_time,
//        genesis_validators_root=pre.genesis_validators_root,
//        slot=pre.slot,
//        fork=Fork(
//            previous_version=pre.fork.current_version,
//            current_version=MERGE_FORK_VERSION,
//            epoch=epoch,
//        ),
//        # History
//        latest_block_header=pre.latest_block_header,
//        block_roots=pre.block_roots,
//        state_roots=pre.state_roots,
//        historical_roots=pre.historical_roots,
//        # Eth1
//        eth1_data=pre.eth1_data,
//        eth1_data_votes=pre.eth1_data_votes,
//        eth1_deposit_index=pre.eth1_deposit_index,
//        # Registry
//        validators=pre.validators,
//        balances=pre.balances,
//        # Randomness
//        randao_mixes=pre.randao_mixes,
//        # Slashings
//        slashings=pre.slashings,
//        # Participation
//        previous_epoch_participation=pre.previous_epoch_participation,
//        current_epoch_participation=pre.current_epoch_participation,
//        # Finality
//        justification_bits=pre.justification_bits,
//        previous_justified_checkpoint=pre.previous_justified_checkpoint,
//        current_justified_checkpoint=pre.current_justified_checkpoint,
//        finalized_checkpoint=pre.finalized_checkpoint,
//        # Inactivity
//        inactivity_scores=pre.inactivity_scores,
//        # Sync
//        current_sync_committee=pre.current_sync_committee,
//        next_sync_committee=pre.next_sync_committee,
//        # Execution-layer
//        latest_execution_payload_header=ExecutionPayloadHeader(),
//    )
//
//    return post
func UpgradeToMerge(ctx context.Context, state state.BeaconState) (state.BeaconState, error) {
	epoch := core.CurrentEpoch(state)

	numValidators := state.NumValidators()
	s := &ethpb.BeaconStateMerge{
		GenesisTime:           state.GenesisTime(),
		GenesisValidatorsRoot: state.GenesisValidatorRoot(),
		Slot:                  state.Slot(),
		Fork: &ethpb.Fork{
			PreviousVersion: state.Fork().CurrentVersion,
			CurrentVersion:  params.BeaconConfig().AltairForkVersion,
			Epoch:           epoch,
		},
		LatestBlockHeader:           state.LatestBlockHeader(),
		BlockRoots:                  state.BlockRoots(),
		StateRoots:                  state.StateRoots(),
		HistoricalRoots:             state.HistoricalRoots(),
		Eth1Data:                    state.Eth1Data(),
		Eth1DataVotes:               state.Eth1DataVotes(),
		Eth1DepositIndex:            state.Eth1DepositIndex(),
		Validators:                  state.Validators(),
		Balances:                    state.Balances(),
		RandaoMixes:                 state.RandaoMixes(),
		Slashings:                   state.Slashings(),
		PreviousEpochParticipation:  make([]byte, numValidators),
		CurrentEpochParticipation:   make([]byte, numValidators),
		JustificationBits:           state.JustificationBits(),
		PreviousJustifiedCheckpoint: state.PreviousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:  state.CurrentJustifiedCheckpoint(),
		FinalizedCheckpoint:         state.FinalizedCheckpoint(),
		InactivityScores:            make([]uint64, numValidators),
		LatestExecutionPayloadHeader: &ethpb.ExecutionPayloadHeader{
			ParentHash:       make([]byte, 32),
			Coinbase:         make([]byte, 20),
			StateRoot:        make([]byte, 32),
			ReceiptRoot:      make([]byte, 32),
			LogsBloom:        make([]byte, 256),
			Random:           make([]byte, 32),
			BlockNumber:      0,
			GasLimit:         0,
			GasUsed:          0,
			Timestamp:        0,
			BaseFeePerGas:    make([]byte, 32),
			BlockHash:        make([]byte, 32),
			TransactionsRoot: make([]byte, 32),
		},
	}

	return v2.InitializeFromProto(s)
}
