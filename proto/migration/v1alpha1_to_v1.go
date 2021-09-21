package migration

import (
	"github.com/pkg/errors"
	statev2 "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"google.golang.org/protobuf/proto"
)

// BlockIfaceToV1BlockHeader converts a signed beacon block interface into a signed beacon block header.
func BlockIfaceToV1BlockHeader(block block.SignedBeaconBlock) (*ethpbv1.SignedBeaconBlockHeader, error) {
	bodyRoot, err := block.Block().Body().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get body root of block")
	}
	return &ethpbv1.SignedBeaconBlockHeader{
		Message: &ethpbv1.BeaconBlockHeader{
			Slot:          block.Block().Slot(),
			ProposerIndex: block.Block().ProposerIndex(),
			ParentRoot:    block.Block().ParentRoot(),
			StateRoot:     block.Block().StateRoot(),
			BodyRoot:      bodyRoot[:],
		},
		Signature: block.Signature(),
	}, nil
}

// V1Alpha1ToV1SignedBlock converts a v1alpha1 SignedBeaconBlock proto to a v1 proto.
func V1Alpha1ToV1SignedBlock(alphaBlk *ethpbalpha.SignedBeaconBlock) (*ethpbv1.SignedBeaconBlock, error) {
	marshaledBlk, err := proto.Marshal(alphaBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1Block := &ethpbv1.SignedBeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1Block, nil
}

// V1ToV1Alpha1SignedBlock converts a v1 SignedBeaconBlock proto to a v1alpha1 proto.
func V1ToV1Alpha1SignedBlock(alphaBlk *ethpbv1.SignedBeaconBlock) (*ethpbalpha.SignedBeaconBlock, error) {
	marshaledBlk, err := proto.Marshal(alphaBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// V1Alpha1ToV1Block converts a v1alpha1 BeaconBlock proto to a v1 proto.
func V1Alpha1ToV1Block(alphaBlk *ethpbalpha.BeaconBlock) (*ethpbv1.BeaconBlock, error) {
	marshaledBlk, err := proto.Marshal(alphaBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1Block := &ethpbv1.BeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1Block, nil
}

// V1Alpha1AggregateAttAndProofToV1 converts a v1alpha1 aggregate attestation and proof to v1.
func V1Alpha1AggregateAttAndProofToV1(v1alpha1Att *ethpbalpha.AggregateAttestationAndProof) *ethpbv1.AggregateAttestationAndProof {
	if v1alpha1Att == nil {
		return &ethpbv1.AggregateAttestationAndProof{}
	}
	return &ethpbv1.AggregateAttestationAndProof{
		AggregatorIndex: v1alpha1Att.AggregatorIndex,
		Aggregate:       V1Alpha1AttestationToV1(v1alpha1Att.Aggregate),
		SelectionProof:  v1alpha1Att.SelectionProof,
	}
}

// V1SignedAggregateAttAndProofToV1Alpha1 converts a v1 signed aggregate attestation and proof to v1alpha1.
func V1SignedAggregateAttAndProofToV1Alpha1(v1Att *ethpbv1.SignedAggregateAttestationAndProof) *ethpbalpha.SignedAggregateAttestationAndProof {
	if v1Att == nil {
		return &ethpbalpha.SignedAggregateAttestationAndProof{}
	}
	return &ethpbalpha.SignedAggregateAttestationAndProof{
		Message: &ethpbalpha.AggregateAttestationAndProof{
			AggregatorIndex: v1Att.Message.AggregatorIndex,
			Aggregate:       V1AttestationToV1Alpha1(v1Att.Message.Aggregate),
			SelectionProof:  v1Att.Message.SelectionProof,
		},
		Signature: v1Att.Signature,
	}
}

// V1Alpha1IndexedAttToV1 converts a v1alpha1 indexed attestation to v1.
func V1Alpha1IndexedAttToV1(v1alpha1Att *ethpbalpha.IndexedAttestation) *ethpbv1.IndexedAttestation {
	if v1alpha1Att == nil {
		return &ethpbv1.IndexedAttestation{}
	}
	return &ethpbv1.IndexedAttestation{
		AttestingIndices: v1alpha1Att.AttestingIndices,
		Data:             V1Alpha1AttDataToV1(v1alpha1Att.Data),
		Signature:        v1alpha1Att.Signature,
	}
}

// V1Alpha1AttestationToV1 converts a v1alpha1 attestation to v1.
func V1Alpha1AttestationToV1(v1alpha1Att *ethpbalpha.Attestation) *ethpbv1.Attestation {
	if v1alpha1Att == nil {
		return &ethpbv1.Attestation{}
	}
	return &ethpbv1.Attestation{
		AggregationBits: v1alpha1Att.AggregationBits,
		Data:            V1Alpha1AttDataToV1(v1alpha1Att.Data),
		Signature:       v1alpha1Att.Signature,
	}
}

// V1AttestationToV1Alpha1 converts a v1 attestation to v1alpha1.
func V1AttestationToV1Alpha1(v1Att *ethpbv1.Attestation) *ethpbalpha.Attestation {
	if v1Att == nil {
		return &ethpbalpha.Attestation{}
	}
	return &ethpbalpha.Attestation{
		AggregationBits: v1Att.AggregationBits,
		Data:            V1AttDataToV1Alpha1(v1Att.Data),
		Signature:       v1Att.Signature,
	}
}

// V1Alpha1AttDataToV1 converts a v1alpha1 attestation data to v1.
func V1Alpha1AttDataToV1(v1alpha1AttData *ethpbalpha.AttestationData) *ethpbv1.AttestationData {
	if v1alpha1AttData == nil || v1alpha1AttData.Source == nil || v1alpha1AttData.Target == nil {
		return &ethpbv1.AttestationData{}
	}
	return &ethpbv1.AttestationData{
		Slot:            v1alpha1AttData.Slot,
		Index:           v1alpha1AttData.CommitteeIndex,
		BeaconBlockRoot: v1alpha1AttData.BeaconBlockRoot,
		Source: &ethpbv1.Checkpoint{
			Root:  v1alpha1AttData.Source.Root,
			Epoch: v1alpha1AttData.Source.Epoch,
		},
		Target: &ethpbv1.Checkpoint{
			Root:  v1alpha1AttData.Target.Root,
			Epoch: v1alpha1AttData.Target.Epoch,
		},
	}
}

// V1Alpha1AttSlashingToV1 converts a v1alpha1 attester slashing to v1.
func V1Alpha1AttSlashingToV1(v1alpha1Slashing *ethpbalpha.AttesterSlashing) *ethpbv1.AttesterSlashing {
	if v1alpha1Slashing == nil {
		return &ethpbv1.AttesterSlashing{}
	}
	return &ethpbv1.AttesterSlashing{
		Attestation_1: V1Alpha1IndexedAttToV1(v1alpha1Slashing.Attestation_1),
		Attestation_2: V1Alpha1IndexedAttToV1(v1alpha1Slashing.Attestation_2),
	}
}

// V1Alpha1SignedHeaderToV1 converts a v1alpha1 signed beacon block header to v1.
func V1Alpha1SignedHeaderToV1(v1alpha1Hdr *ethpbalpha.SignedBeaconBlockHeader) *ethpbv1.SignedBeaconBlockHeader {
	if v1alpha1Hdr == nil || v1alpha1Hdr.Header == nil {
		return &ethpbv1.SignedBeaconBlockHeader{}
	}
	return &ethpbv1.SignedBeaconBlockHeader{
		Message: &ethpbv1.BeaconBlockHeader{
			Slot:          v1alpha1Hdr.Header.Slot,
			ProposerIndex: v1alpha1Hdr.Header.ProposerIndex,
			ParentRoot:    v1alpha1Hdr.Header.ParentRoot,
			StateRoot:     v1alpha1Hdr.Header.StateRoot,
			BodyRoot:      v1alpha1Hdr.Header.BodyRoot,
		},
		Signature: v1alpha1Hdr.Signature,
	}
}

// V1SignedHeaderToV1Alpha1 converts a v1 signed beacon block header to v1alpha1.
func V1SignedHeaderToV1Alpha1(v1Header *ethpbv1.SignedBeaconBlockHeader) *ethpbalpha.SignedBeaconBlockHeader {
	if v1Header == nil || v1Header.Message == nil {
		return &ethpbalpha.SignedBeaconBlockHeader{}
	}
	return &ethpbalpha.SignedBeaconBlockHeader{
		Header: &ethpbalpha.BeaconBlockHeader{
			Slot:          v1Header.Message.Slot,
			ProposerIndex: v1Header.Message.ProposerIndex,
			ParentRoot:    v1Header.Message.ParentRoot,
			StateRoot:     v1Header.Message.StateRoot,
			BodyRoot:      v1Header.Message.BodyRoot,
		},
		Signature: v1Header.Signature,
	}
}

// V1Alpha1ProposerSlashingToV1 converts a v1alpha1 proposer slashing to v1.
func V1Alpha1ProposerSlashingToV1(v1alpha1Slashing *ethpbalpha.ProposerSlashing) *ethpbv1.ProposerSlashing {
	if v1alpha1Slashing == nil {
		return &ethpbv1.ProposerSlashing{}
	}
	return &ethpbv1.ProposerSlashing{
		SignedHeader_1: V1Alpha1SignedHeaderToV1(v1alpha1Slashing.Header_1),
		SignedHeader_2: V1Alpha1SignedHeaderToV1(v1alpha1Slashing.Header_2),
	}
}

// V1Alpha1ExitToV1 converts a v1alpha1 SignedVoluntaryExit to v1.
func V1Alpha1ExitToV1(v1alpha1Exit *ethpbalpha.SignedVoluntaryExit) *ethpbv1.SignedVoluntaryExit {
	if v1alpha1Exit == nil || v1alpha1Exit.Exit == nil {
		return &ethpbv1.SignedVoluntaryExit{}
	}
	return &ethpbv1.SignedVoluntaryExit{
		Message: &ethpbv1.VoluntaryExit{
			Epoch:          v1alpha1Exit.Exit.Epoch,
			ValidatorIndex: v1alpha1Exit.Exit.ValidatorIndex,
		},
		Signature: v1alpha1Exit.Signature,
	}
}

// V1ExitToV1Alpha1 converts a v1 SignedVoluntaryExit to v1alpha1.
func V1ExitToV1Alpha1(v1Exit *ethpbv1.SignedVoluntaryExit) *ethpbalpha.SignedVoluntaryExit {
	if v1Exit == nil || v1Exit.Message == nil {
		return &ethpbalpha.SignedVoluntaryExit{}
	}
	return &ethpbalpha.SignedVoluntaryExit{
		Exit: &ethpbalpha.VoluntaryExit{
			Epoch:          v1Exit.Message.Epoch,
			ValidatorIndex: v1Exit.Message.ValidatorIndex,
		},
		Signature: v1Exit.Signature,
	}
}

// V1AttToV1Alpha1 converts a v1 attestation to v1alpha1.
func V1AttToV1Alpha1(v1Att *ethpbv1.Attestation) *ethpbalpha.Attestation {
	if v1Att == nil {
		return &ethpbalpha.Attestation{}
	}
	return &ethpbalpha.Attestation{
		AggregationBits: v1Att.AggregationBits,
		Data:            V1AttDataToV1Alpha1(v1Att.Data),
		Signature:       v1Att.Signature,
	}
}

// V1IndexedAttToV1Alpha1 converts a v1 indexed attestation to v1alpha1.
func V1IndexedAttToV1Alpha1(v1Att *ethpbv1.IndexedAttestation) *ethpbalpha.IndexedAttestation {
	if v1Att == nil {
		return &ethpbalpha.IndexedAttestation{}
	}
	return &ethpbalpha.IndexedAttestation{
		AttestingIndices: v1Att.AttestingIndices,
		Data:             V1AttDataToV1Alpha1(v1Att.Data),
		Signature:        v1Att.Signature,
	}
}

// V1AttDataToV1Alpha1 converts a v1 attestation data to v1alpha1.
func V1AttDataToV1Alpha1(v1AttData *ethpbv1.AttestationData) *ethpbalpha.AttestationData {
	if v1AttData == nil || v1AttData.Source == nil || v1AttData.Target == nil {
		return &ethpbalpha.AttestationData{}
	}
	return &ethpbalpha.AttestationData{
		Slot:            v1AttData.Slot,
		CommitteeIndex:  v1AttData.Index,
		BeaconBlockRoot: v1AttData.BeaconBlockRoot,
		Source: &ethpbalpha.Checkpoint{
			Root:  v1AttData.Source.Root,
			Epoch: v1AttData.Source.Epoch,
		},
		Target: &ethpbalpha.Checkpoint{
			Root:  v1AttData.Target.Root,
			Epoch: v1AttData.Target.Epoch,
		},
	}
}

// V1AttSlashingToV1Alpha1 converts a v1 attester slashing to v1alpha1.
func V1AttSlashingToV1Alpha1(v1Slashing *ethpbv1.AttesterSlashing) *ethpbalpha.AttesterSlashing {
	if v1Slashing == nil {
		return &ethpbalpha.AttesterSlashing{}
	}
	return &ethpbalpha.AttesterSlashing{
		Attestation_1: V1IndexedAttToV1Alpha1(v1Slashing.Attestation_1),
		Attestation_2: V1IndexedAttToV1Alpha1(v1Slashing.Attestation_2),
	}
}

// V1ProposerSlashingToV1Alpha1 converts a v1 proposer slashing to v1alpha1.
func V1ProposerSlashingToV1Alpha1(v1Slashing *ethpbv1.ProposerSlashing) *ethpbalpha.ProposerSlashing {
	if v1Slashing == nil {
		return &ethpbalpha.ProposerSlashing{}
	}
	return &ethpbalpha.ProposerSlashing{
		Header_1: V1SignedHeaderToV1Alpha1(v1Slashing.SignedHeader_1),
		Header_2: V1SignedHeaderToV1Alpha1(v1Slashing.SignedHeader_2),
	}
}

// V1Alpha1ValidatorToV1 converts a v1alpha1 validator to v1.
func V1Alpha1ValidatorToV1(v1Alpha1Validator *ethpbalpha.Validator) *ethpbv1.Validator {
	if v1Alpha1Validator == nil {
		return &ethpbv1.Validator{}
	}
	return &ethpbv1.Validator{
		Pubkey:                     v1Alpha1Validator.PublicKey,
		WithdrawalCredentials:      v1Alpha1Validator.WithdrawalCredentials,
		EffectiveBalance:           v1Alpha1Validator.EffectiveBalance,
		Slashed:                    v1Alpha1Validator.Slashed,
		ActivationEligibilityEpoch: v1Alpha1Validator.ActivationEligibilityEpoch,
		ActivationEpoch:            v1Alpha1Validator.ActivationEpoch,
		ExitEpoch:                  v1Alpha1Validator.ExitEpoch,
		WithdrawableEpoch:          v1Alpha1Validator.WithdrawableEpoch,
	}
}

// V1ValidatorToV1Alpha1 converts a v1 validator to v1alpha1.
func V1ValidatorToV1Alpha1(v1Validator *ethpbv1.Validator) *ethpbalpha.Validator {
	if v1Validator == nil {
		return &ethpbalpha.Validator{}
	}
	return &ethpbalpha.Validator{
		PublicKey:                  v1Validator.Pubkey,
		WithdrawalCredentials:      v1Validator.WithdrawalCredentials,
		EffectiveBalance:           v1Validator.EffectiveBalance,
		Slashed:                    v1Validator.Slashed,
		ActivationEligibilityEpoch: v1Validator.ActivationEligibilityEpoch,
		ActivationEpoch:            v1Validator.ActivationEpoch,
		ExitEpoch:                  v1Validator.ExitEpoch,
		WithdrawableEpoch:          v1Validator.WithdrawableEpoch,
	}
}

// SignedBeaconBlock converts a signed beacon block interface to a v1alpha1 block.
func SignedBeaconBlock(block block.SignedBeaconBlock) (*ethpbv1.SignedBeaconBlock, error) {
	if block == nil || block.IsNil() {
		return nil, errors.New("could not find requested block")
	}
	blk, err := block.PbPhase0Block()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get raw block")
	}

	v1Block, err := V1Alpha1ToV1SignedBlock(blk)
	if err != nil {
		return nil, errors.New("could not convert block to v1 block")
	}

	return v1Block, nil
}

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

func BeaconStateAltairToV2(altairState *statev2.BeaconState) (*ethpbv2.BeaconStateV2, error) {
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
		GenesisValidatorsRoot: bytesutil.SafeCopyBytes(altairState.GenesisValidatorRoot()),
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
