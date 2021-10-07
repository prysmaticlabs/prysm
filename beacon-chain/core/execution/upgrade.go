package execution

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v3"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// UpgradeToMerge updates input state to return the version Merge state.
func UpgradeToMerge(ctx context.Context, state state.BeaconState) (state.BeaconState, error) {
	epoch := time.CurrentEpoch(state)

	numValidators := state.NumValidators()
	s := &ethpb.BeaconStateMerge{
		GenesisTime:           state.GenesisTime(),
		GenesisValidatorsRoot: state.GenesisValidatorRoot(),
		Slot:                  state.Slot(),
		Fork: &ethpb.Fork{
			PreviousVersion: state.Fork().CurrentVersion,
			CurrentVersion:  params.BeaconConfig().MergeForkVersion,
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
