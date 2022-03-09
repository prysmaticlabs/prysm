package migration

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// V1Alpha1BeaconBlockAltairToV2 converts a v1alpha1 Altair beacon block to a v2 Altair block.
func V1Alpha1BeaconBlockAltairToV2(v1alpha1Block *ethpbalpha.BeaconBlockAltair) (*ethpbv2.BeaconBlockAltair, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BeaconBlockAltair{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// AltairToV1Alpha1SignedBlock converts a v2 SignedBeaconBlockAltair proto to a v1alpha1 proto.
func AltairToV1Alpha1SignedBlock(altairBlk *ethpbv2.SignedBeaconBlockAltair) (*ethpbalpha.SignedBeaconBlockAltair, error) {
	marshaledBlk, err := proto.Marshal(altairBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBeaconBlockAltair{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// BellatrixToV1Alpha1SignedBlock converts a v2 SignedBeaconBlockBellatrix proto to a v1alpha1 proto.
func BellatrixToV1Alpha1SignedBlock(bellatrixBlk *ethpbv2.SignedBeaconBlockBellatrix) (*ethpbalpha.SignedBeaconBlockBellatrix, error) {
	marshaledBlk, err := proto.Marshal(bellatrixBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBeaconBlockBellatrix{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// V1Alpha1BeaconBlockBellatrixToV2 converts a v1alpha1 Bellatrix beacon block to a v2
// Bellatrix block.
func V1Alpha1BeaconBlockBellatrixToV2(v1alpha1Block *ethpbalpha.BeaconBlockBellatrix) (*ethpbv2.BeaconBlockBellatrix, error) {
	marshaledBlk, err := proto.Marshal(v1alpha1Block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v2Block := &ethpbv2.BeaconBlockBellatrix{}
	if err := proto.Unmarshal(marshaledBlk, v2Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v2Block, nil
}

// BeaconStateAltairToProto converts a state.BeaconStateAltair object to its protobuf equivalent.
func BeaconStateAltairToProto(altairState state.BeaconStateAltair) (*ethpbv2.BeaconStateV2, error) {
	sourceFork := altairState.Fork()
	sourceLatestBlockHeader := altairState.LatestBlockHeader()
	sourceEth1Data := altairState.Eth1Data()
	sourceEth1DataVotes := altairState.Eth1DataVotes()
	sourceValidators := altairState.Validators()
	sourcePrevJustifiedCheckpoint := altairState.PreviousJustifiedCheckpoint()
	sourceCurrJustifiedCheckpoint := altairState.CurrentJustifiedCheckpoint()
	sourceFinalizedCheckpoint := altairState.FinalizedCheckpoint()

	resultEth1DataVotes := make([]*ethpbv1.Eth1Data, len(sourceEth1DataVotes))
	for i, vote := range sourceEth1DataVotes {
		resultEth1DataVotes[i] = &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(vote.DepositRoot),
			DepositCount: vote.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(vote.BlockHash),
		}
	}
	resultValidators := make([]*ethpbv1.Validator, len(sourceValidators))
	for i, validator := range sourceValidators {
		resultValidators[i] = &ethpbv1.Validator{
			Pubkey:                     bytesutil.SafeCopyBytes(validator.PublicKey),
			WithdrawalCredentials:      bytesutil.SafeCopyBytes(validator.WithdrawalCredentials),
			EffectiveBalance:           validator.EffectiveBalance,
			Slashed:                    validator.Slashed,
			ActivationEligibilityEpoch: validator.ActivationEligibilityEpoch,
			ActivationEpoch:            validator.ActivationEpoch,
			ExitEpoch:                  validator.ExitEpoch,
			WithdrawableEpoch:          validator.WithdrawableEpoch,
		}
	}

	sourcePrevEpochParticipation, err := altairState.PreviousEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get previous epoch participation")
	}
	sourceCurrEpochParticipation, err := altairState.CurrentEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current epoch participation")
	}
	sourceInactivityScores, err := altairState.InactivityScores()
	if err != nil {
		return nil, errors.Wrap(err, "could not get inactivity scores")
	}
	sourceCurrSyncCommittee, err := altairState.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}
	sourceNextSyncCommittee, err := altairState.NextSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next sync committee")
	}

	result := &ethpbv2.BeaconStateV2{
		GenesisTime:           altairState.GenesisTime(),
		GenesisValidatorsRoot: bytesutil.SafeCopyBytes(altairState.GenesisValidatorsRoot()),
		Slot:                  altairState.Slot(),
		Fork: &ethpbv1.Fork{
			PreviousVersion: bytesutil.SafeCopyBytes(sourceFork.PreviousVersion),
			CurrentVersion:  bytesutil.SafeCopyBytes(sourceFork.CurrentVersion),
			Epoch:           sourceFork.Epoch,
		},
		LatestBlockHeader: &ethpbv1.BeaconBlockHeader{
			Slot:          sourceLatestBlockHeader.Slot,
			ProposerIndex: sourceLatestBlockHeader.ProposerIndex,
			ParentRoot:    bytesutil.SafeCopyBytes(sourceLatestBlockHeader.ParentRoot),
			StateRoot:     bytesutil.SafeCopyBytes(sourceLatestBlockHeader.StateRoot),
			BodyRoot:      bytesutil.SafeCopyBytes(sourceLatestBlockHeader.BodyRoot),
		},
		BlockRoots:      bytesutil.SafeCopy2dBytes(altairState.BlockRoots()),
		StateRoots:      bytesutil.SafeCopy2dBytes(altairState.StateRoots()),
		HistoricalRoots: bytesutil.SafeCopy2dBytes(altairState.HistoricalRoots()),
		Eth1Data: &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(sourceEth1Data.DepositRoot),
			DepositCount: sourceEth1Data.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(sourceEth1Data.BlockHash),
		},
		Eth1DataVotes:              resultEth1DataVotes,
		Eth1DepositIndex:           altairState.Eth1DepositIndex(),
		Validators:                 resultValidators,
		Balances:                   altairState.Balances(),
		RandaoMixes:                bytesutil.SafeCopy2dBytes(altairState.RandaoMixes()),
		Slashings:                  altairState.Slashings(),
		PreviousEpochParticipation: bytesutil.SafeCopyBytes(sourcePrevEpochParticipation),
		CurrentEpochParticipation:  bytesutil.SafeCopyBytes(sourceCurrEpochParticipation),
		JustificationBits:          bytesutil.SafeCopyBytes(altairState.JustificationBits()),
		PreviousJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourcePrevJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourcePrevJustifiedCheckpoint.Root),
		},
		CurrentJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceCurrJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceCurrJustifiedCheckpoint.Root),
		},
		FinalizedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceFinalizedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceFinalizedCheckpoint.Root),
		},
		InactivityScores: sourceInactivityScores,
		CurrentSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceCurrSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceCurrSyncCommittee.AggregatePubkey),
		},
		NextSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceNextSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceNextSyncCommittee.AggregatePubkey),
		},
	}

	return result, nil
}

// BeaconStateBellatrixToProto converts a state.BeaconStateBellatrix object to its protobuf equivalent.
func BeaconStateBellatrixToProto(st state.BeaconStateBellatrix) (*ethpbv2.BeaconStateBellatrix, error) {
	sourceFork := st.Fork()
	sourceLatestBlockHeader := st.LatestBlockHeader()
	sourceEth1Data := st.Eth1Data()
	sourceEth1DataVotes := st.Eth1DataVotes()
	sourceValidators := st.Validators()
	sourcePrevJustifiedCheckpoint := st.PreviousJustifiedCheckpoint()
	sourceCurrJustifiedCheckpoint := st.CurrentJustifiedCheckpoint()
	sourceFinalizedCheckpoint := st.FinalizedCheckpoint()

	resultEth1DataVotes := make([]*ethpbv1.Eth1Data, len(sourceEth1DataVotes))
	for i, vote := range sourceEth1DataVotes {
		resultEth1DataVotes[i] = &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(vote.DepositRoot),
			DepositCount: vote.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(vote.BlockHash),
		}
	}
	resultValidators := make([]*ethpbv1.Validator, len(sourceValidators))
	for i, validator := range sourceValidators {
		resultValidators[i] = &ethpbv1.Validator{
			Pubkey:                     bytesutil.SafeCopyBytes(validator.PublicKey),
			WithdrawalCredentials:      bytesutil.SafeCopyBytes(validator.WithdrawalCredentials),
			EffectiveBalance:           validator.EffectiveBalance,
			Slashed:                    validator.Slashed,
			ActivationEligibilityEpoch: validator.ActivationEligibilityEpoch,
			ActivationEpoch:            validator.ActivationEpoch,
			ExitEpoch:                  validator.ExitEpoch,
			WithdrawableEpoch:          validator.WithdrawableEpoch,
		}
	}

	sourcePrevEpochParticipation, err := st.PreviousEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get previous epoch participation")
	}
	sourceCurrEpochParticipation, err := st.CurrentEpochParticipation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current epoch participation")
	}
	sourceInactivityScores, err := st.InactivityScores()
	if err != nil {
		return nil, errors.Wrap(err, "could not get inactivity scores")
	}
	sourceCurrSyncCommittee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get current sync committee")
	}
	sourceNextSyncCommittee, err := st.NextSyncCommittee()
	if err != nil {
		return nil, errors.Wrap(err, "could not get next sync committee")
	}
	sourceLatestExecutionPaylodHeader, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, errors.Wrap(err, "could not get latest execution payload header")
	}

	result := &ethpbv2.BeaconStateBellatrix{
		GenesisTime:           st.GenesisTime(),
		GenesisValidatorsRoot: bytesutil.SafeCopyBytes(st.GenesisValidatorsRoot()),
		Slot:                  st.Slot(),
		Fork: &ethpbv1.Fork{
			PreviousVersion: bytesutil.SafeCopyBytes(sourceFork.PreviousVersion),
			CurrentVersion:  bytesutil.SafeCopyBytes(sourceFork.CurrentVersion),
			Epoch:           sourceFork.Epoch,
		},
		LatestBlockHeader: &ethpbv1.BeaconBlockHeader{
			Slot:          sourceLatestBlockHeader.Slot,
			ProposerIndex: sourceLatestBlockHeader.ProposerIndex,
			ParentRoot:    bytesutil.SafeCopyBytes(sourceLatestBlockHeader.ParentRoot),
			StateRoot:     bytesutil.SafeCopyBytes(sourceLatestBlockHeader.StateRoot),
			BodyRoot:      bytesutil.SafeCopyBytes(sourceLatestBlockHeader.BodyRoot),
		},
		BlockRoots:      bytesutil.SafeCopy2dBytes(st.BlockRoots()),
		StateRoots:      bytesutil.SafeCopy2dBytes(st.StateRoots()),
		HistoricalRoots: bytesutil.SafeCopy2dBytes(st.HistoricalRoots()),
		Eth1Data: &ethpbv1.Eth1Data{
			DepositRoot:  bytesutil.SafeCopyBytes(sourceEth1Data.DepositRoot),
			DepositCount: sourceEth1Data.DepositCount,
			BlockHash:    bytesutil.SafeCopyBytes(sourceEth1Data.BlockHash),
		},
		Eth1DataVotes:              resultEth1DataVotes,
		Eth1DepositIndex:           st.Eth1DepositIndex(),
		Validators:                 resultValidators,
		Balances:                   st.Balances(),
		RandaoMixes:                bytesutil.SafeCopy2dBytes(st.RandaoMixes()),
		Slashings:                  st.Slashings(),
		PreviousEpochParticipation: bytesutil.SafeCopyBytes(sourcePrevEpochParticipation),
		CurrentEpochParticipation:  bytesutil.SafeCopyBytes(sourceCurrEpochParticipation),
		JustificationBits:          bytesutil.SafeCopyBytes(st.JustificationBits()),
		PreviousJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourcePrevJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourcePrevJustifiedCheckpoint.Root),
		},
		CurrentJustifiedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceCurrJustifiedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceCurrJustifiedCheckpoint.Root),
		},
		FinalizedCheckpoint: &ethpbv1.Checkpoint{
			Epoch: sourceFinalizedCheckpoint.Epoch,
			Root:  bytesutil.SafeCopyBytes(sourceFinalizedCheckpoint.Root),
		},
		InactivityScores: sourceInactivityScores,
		CurrentSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceCurrSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceCurrSyncCommittee.AggregatePubkey),
		},
		NextSyncCommittee: &ethpbv2.SyncCommittee{
			Pubkeys:         bytesutil.SafeCopy2dBytes(sourceNextSyncCommittee.Pubkeys),
			AggregatePubkey: bytesutil.SafeCopyBytes(sourceNextSyncCommittee.AggregatePubkey),
		},
		LatestExecutionPayloadHeader: &ethpbv2.ExecutionPayloadHeader{
			ParentHash:       bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.ParentHash),
			FeeRecipient:     bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.FeeRecipient),
			StateRoot:        bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.StateRoot),
			ReceiptRoot:      bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.ReceiptRoot),
			LogsBloom:        bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.LogsBloom),
			PrevRandao:       bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.PrevRandao),
			BlockNumber:      sourceLatestExecutionPaylodHeader.BlockNumber,
			GasLimit:         sourceLatestExecutionPaylodHeader.GasLimit,
			GasUsed:          sourceLatestExecutionPaylodHeader.GasUsed,
			Timestamp:        sourceLatestExecutionPaylodHeader.Timestamp,
			ExtraData:        bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.ExtraData),
			BaseFeePerGas:    bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.BaseFeePerGas),
			BlockHash:        bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.BlockHash),
			TransactionsRoot: bytesutil.SafeCopyBytes(sourceLatestExecutionPaylodHeader.TransactionsRoot),
		},
	}

	return result, nil
}

func V1Alpha1SignedContributionAndProofToV2(alphaContribution *ethpbalpha.SignedContributionAndProof) *ethpbv2.SignedContributionAndProof {
	result := &ethpbv2.SignedContributionAndProof{
		Message: &ethpbv2.ContributionAndProof{
			AggregatorIndex: alphaContribution.Message.AggregatorIndex,
			Contribution: &ethpbv2.SyncCommitteeContribution{
				Slot:              alphaContribution.Message.Contribution.Slot,
				BeaconBlockRoot:   alphaContribution.Message.Contribution.BlockRoot,
				SubcommitteeIndex: alphaContribution.Message.Contribution.SubcommitteeIndex,
				AggregationBits:   alphaContribution.Message.Contribution.AggregationBits,
				Signature:         alphaContribution.Message.Contribution.Signature,
			},
			SelectionProof: alphaContribution.Message.SelectionProof,
		},
		Signature: alphaContribution.Signature,
	}
	return result
}
