package capella

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/state/types"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// UpgradeToCapella updates a generic state to return the version Capella state.
func UpgradeToCapella(st types.BeaconState) (types.BeaconState, error) {
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
	payloadHeader, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	txRoot, err := payloadHeader.TransactionsRoot()
	if err != nil {
		return nil, err
	}

	hrs, err := st.HistoricalRoots()
	if err != nil {
		return nil, err
	}
	s := &ethpb.BeaconStateCapella{
		GenesisTime:           st.GenesisTime(),
		GenesisValidatorsRoot: st.GenesisValidatorsRoot(),
		Slot:                  st.Slot(),
		Fork: &ethpb.Fork{
			PreviousVersion: st.Fork().CurrentVersion,
			CurrentVersion:  params.BeaconConfig().CapellaForkVersion,
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
		LatestExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderCapella{
			ParentHash:       payloadHeader.ParentHash(),
			FeeRecipient:     payloadHeader.FeeRecipient(),
			StateRoot:        payloadHeader.StateRoot(),
			ReceiptsRoot:     payloadHeader.ReceiptsRoot(),
			LogsBloom:        payloadHeader.LogsBloom(),
			PrevRandao:       payloadHeader.PrevRandao(),
			BlockNumber:      payloadHeader.BlockNumber(),
			GasLimit:         payloadHeader.GasLimit(),
			GasUsed:          payloadHeader.GasUsed(),
			Timestamp:        payloadHeader.Timestamp(),
			ExtraData:        payloadHeader.ExtraData(),
			BaseFeePerGas:    payloadHeader.BaseFeePerGas(),
			BlockHash:        payloadHeader.BlockHash(),
			TransactionsRoot: txRoot,
			WithdrawalsRoot:  make([]byte, 32),
		},
		NextWithdrawalIndex:          0,
		NextWithdrawalValidatorIndex: 0,
		HistoricalSummaries:          make([]*ethpb.HistoricalSummary, 0),
	}

	return state.InitializeFromProtoUnsafeCapella(s)
}
