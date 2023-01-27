package execution

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// UpgradeToBellatrix updates inputs a generic state to return the version Bellatrix state.
// It inserts an empty `ExecutionPayloadHeader` into the state.
func UpgradeToBellatrix(st types.BeaconState) (types.BeaconState, error) {
	epoch := time.CurrentEpoch(st)

	currentSyncCommittee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, err
	}
	nextSyncCommittee, err := st.NextSyncCommittee()
	if err != nil {
		return nil, err
	}
	prevEpochParticipation, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, err
	}
	currentEpochParticipation, err := st.CurrentEpochParticipation()
	if err != nil {
		return nil, err
	}
	inactivityScores, err := st.InactivityScores()
	if err != nil {
		return nil, err
	}

	hrs, err := st.HistoricalRoots()
	if err != nil {
		return nil, err
	}
	s := &ethpb.BeaconStateBellatrix{
		GenesisTime:           st.GenesisTime(),
		GenesisValidatorsRoot: st.GenesisValidatorsRoot(),
		Slot:                  st.Slot(),
		Fork: &ethpb.Fork{
			PreviousVersion: st.Fork().CurrentVersion,
			CurrentVersion:  params.BeaconConfig().BellatrixForkVersion,
			Epoch:           epoch,
		},
		LatestBlockHeader:           st.LatestBlockHeader(),
		BlockRoots:                  st.BlockRoots(),
		StateRoots:                  st.StateRoots(),
		HistoricalRoots:             hrs,
		Eth1Data:                    st.Eth1Data(),
		Eth1DataVotes:               st.Eth1DataVotes(),
		Eth1DepositIndex:            st.Eth1DepositIndex(),
		Validators:                  st.Validators(),
		Balances:                    st.Balances(),
		RandaoMixes:                 st.RandaoMixes(),
		Slashings:                   st.Slashings(),
		PreviousEpochParticipation:  prevEpochParticipation,
		CurrentEpochParticipation:   currentEpochParticipation,
		JustificationBits:           st.JustificationBits(),
		PreviousJustifiedCheckpoint: st.PreviousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:  st.CurrentJustifiedCheckpoint(),
		FinalizedCheckpoint:         st.FinalizedCheckpoint(),
		InactivityScores:            inactivityScores,
		CurrentSyncCommittee:        currentSyncCommittee,
		NextSyncCommittee:           nextSyncCommittee,
		LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{
			ParentHash:       make([]byte, 32),
			FeeRecipient:     make([]byte, 20),
			StateRoot:        make([]byte, 32),
			ReceiptsRoot:     make([]byte, 32),
			LogsBloom:        make([]byte, 256),
			PrevRandao:       make([]byte, 32),
			BlockNumber:      0,
			GasLimit:         0,
			GasUsed:          0,
			Timestamp:        0,
			BaseFeePerGas:    make([]byte, 32),
			BlockHash:        make([]byte, 32),
			TransactionsRoot: make([]byte, 32),
		},
	}

	return state.InitializeFromProtoUnsafeBellatrix(s)
}
