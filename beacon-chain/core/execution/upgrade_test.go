package execution_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/execution"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
)

func TestUpgradeToMerge(t *testing.T) {
	st, _ := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	preForkState := st.Copy()
	mSt, err := execution.UpgradeToMerge(context.Background(), st)
	require.NoError(t, err)

	require.Equal(t, preForkState.GenesisTime(), mSt.GenesisTime())
	require.DeepSSZEqual(t, preForkState.GenesisValidatorRoot(), mSt.GenesisValidatorRoot())
	require.Equal(t, preForkState.Slot(), mSt.Slot())
	require.DeepSSZEqual(t, preForkState.LatestBlockHeader(), mSt.LatestBlockHeader())
	require.DeepSSZEqual(t, preForkState.BlockRoots(), mSt.BlockRoots())
	require.DeepSSZEqual(t, preForkState.StateRoots(), mSt.StateRoots())
	require.DeepSSZEqual(t, preForkState.HistoricalRoots(), mSt.HistoricalRoots())
	require.DeepSSZEqual(t, preForkState.Eth1Data(), mSt.Eth1Data())
	require.DeepSSZEqual(t, preForkState.Eth1DataVotes(), mSt.Eth1DataVotes())
	require.DeepSSZEqual(t, preForkState.Eth1DepositIndex(), mSt.Eth1DepositIndex())
	require.DeepSSZEqual(t, preForkState.Validators(), mSt.Validators())
	require.DeepSSZEqual(t, preForkState.Balances(), mSt.Balances())
	require.DeepSSZEqual(t, preForkState.RandaoMixes(), mSt.RandaoMixes())
	require.DeepSSZEqual(t, preForkState.Slashings(), mSt.Slashings())
	require.DeepSSZEqual(t, preForkState.JustificationBits(), mSt.JustificationBits())
	require.DeepSSZEqual(t, preForkState.PreviousJustifiedCheckpoint(), mSt.PreviousJustifiedCheckpoint())
	require.DeepSSZEqual(t, preForkState.CurrentJustifiedCheckpoint(), mSt.CurrentJustifiedCheckpoint())
	require.DeepSSZEqual(t, preForkState.FinalizedCheckpoint(), mSt.FinalizedCheckpoint())
	numValidators := mSt.NumValidators()
	p, err := mSt.PreviousEpochParticipation()
	require.NoError(t, err)
	require.DeepSSZEqual(t, make([]byte, numValidators), p)
	p, err = mSt.CurrentEpochParticipation()
	require.NoError(t, err)
	require.DeepSSZEqual(t, make([]byte, numValidators), p)
	s, err := mSt.InactivityScores()
	require.NoError(t, err)
	require.DeepSSZEqual(t, make([]uint64, numValidators), s)

	f := mSt.Fork()
	require.DeepSSZEqual(t, &ethpb.Fork{
		PreviousVersion: st.Fork().CurrentVersion,
		CurrentVersion:  params.BeaconConfig().BellatrixForkVersion,
		Epoch:           time.CurrentEpoch(st),
	}, f)
	csc, err := mSt.CurrentSyncCommittee()
	require.NoError(t, err)
	nsc, err := mSt.NextSyncCommittee()
	require.NoError(t, err)
	require.DeepSSZEqual(t, nsc, csc)

	header, err := mSt.LatestExecutionPayloadHeader()
	require.NoError(t, err)
	wanted := &ethpb.ExecutionPayloadHeader{
		ParentHash:       make([]byte, 32),
		FeeRecipient:     make([]byte, 20),
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
	}
	require.DeepEqual(t, wanted, header)
}
