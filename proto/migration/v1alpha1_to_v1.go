package migration

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"google.golang.org/protobuf/proto"
)

// BlockIfaceToV1BlockHeader converts a signed beacon block interface into a signed beacon block header.
func BlockIfaceToV1BlockHeader(block block.SignedBeaconBlock) (*ethpb.SignedBeaconBlockHeader, error) {
	bodyRoot, err := block.Block().Body().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get body root of block")
	}
	return &ethpb.SignedBeaconBlockHeader{
		Message: &ethpb.BeaconBlockHeader{
			Slot:          block.Block().Slot(),
			ProposerIndex: block.Block().ProposerIndex(),
			ParentRoot:    block.Block().ParentRoot(),
			StateRoot:     block.Block().StateRoot(),
			BodyRoot:      bodyRoot[:],
		},
		Signature: block.Signature(),
	}, nil
}

// V1Alpha1BlockToV1BlockHeader converts a v1alpha1 SignedBeaconBlock proto to a v1 SignedBeaconBlockHeader proto.
func V1Alpha1BlockToV1BlockHeader(block *eth.SignedBeaconBlock) (*ethpb.SignedBeaconBlockHeader, error) {
	bodyRoot, err := block.Block.Body.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get body root of block")
	}
	return &ethpb.SignedBeaconBlockHeader{
		Message: &ethpb.BeaconBlockHeader{
			Slot:          block.Block.Slot,
			ProposerIndex: block.Block.ProposerIndex,
			ParentRoot:    block.Block.ParentRoot,
			StateRoot:     block.Block.StateRoot,
			BodyRoot:      bodyRoot[:],
		},
		Signature: block.Signature,
	}, nil
}

// V1Alpha1ToV1SignedBlock converts a v1alpha1 SignedBeaconBlock proto to a v1 proto.
func V1Alpha1ToV1SignedBlock(alphaBlk *eth.SignedBeaconBlock) (*ethpb.SignedBeaconBlock, error) {
	marshaledBlk, err := proto.Marshal(alphaBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1Block := &ethpb.SignedBeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1Block, nil
}

// V1ToV1Alpha1SignedBlock converts a v1 SignedBeaconBlock proto to a v1alpha1 proto.
func V1ToV1Alpha1SignedBlock(alphaBlk *ethpb.SignedBeaconBlock) (*eth.SignedBeaconBlock, error) {
	marshaledBlk, err := proto.Marshal(alphaBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &eth.SignedBeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// V1Alpha1ToV1Block converts a v1alpha1 BeaconBlock proto to a v1 proto.
func V1Alpha1ToV1Block(alphaBlk *eth.BeaconBlock) (*ethpb.BeaconBlock, error) {
	marshaledBlk, err := proto.Marshal(alphaBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1Block := &ethpb.BeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1Block, nil
}

// V1Alpha1AggregateAttAndProofToV1 converts a v1alpha1 aggregate attestation and proof to v1.
func V1Alpha1AggregateAttAndProofToV1(v1alpha1Att *eth.AggregateAttestationAndProof) *ethpb.AggregateAttestationAndProof {
	if v1alpha1Att == nil {
		return &ethpb.AggregateAttestationAndProof{}
	}
	return &ethpb.AggregateAttestationAndProof{
		AggregatorIndex: v1alpha1Att.AggregatorIndex,
		Aggregate:       V1Alpha1AttestationToV1(v1alpha1Att.Aggregate),
		SelectionProof:  v1alpha1Att.SelectionProof,
	}
}

// V1SignedAggregateAttAndProofToV1Alpha1 converts a v1 signed aggregate attestation and proof to v1alpha1.
func V1SignedAggregateAttAndProofToV1Alpha1(v1Att *ethpb.SignedAggregateAttestationAndProof) *eth.SignedAggregateAttestationAndProof {
	if v1Att == nil {
		return &eth.SignedAggregateAttestationAndProof{}
	}
	return &eth.SignedAggregateAttestationAndProof{
		Message: &eth.AggregateAttestationAndProof{
			AggregatorIndex: v1Att.Message.AggregatorIndex,
			Aggregate:       V1AttestationToV1Alpha1(v1Att.Message.Aggregate),
			SelectionProof:  v1Att.Message.SelectionProof,
		},
		Signature: v1Att.Signature,
	}
}

// V1Alpha1IndexedAttToV1 converts a v1alpha1 indexed attestation to v1.
func V1Alpha1IndexedAttToV1(v1alpha1Att *eth.IndexedAttestation) *ethpb.IndexedAttestation {
	if v1alpha1Att == nil {
		return &ethpb.IndexedAttestation{}
	}
	return &ethpb.IndexedAttestation{
		AttestingIndices: v1alpha1Att.AttestingIndices,
		Data:             V1Alpha1AttDataToV1(v1alpha1Att.Data),
		Signature:        v1alpha1Att.Signature,
	}
}

// V1Alpha1AttestationToV1 converts a v1alpha1 attestation to v1.
func V1Alpha1AttestationToV1(v1alpha1Att *eth.Attestation) *ethpb.Attestation {
	if v1alpha1Att == nil {
		return &ethpb.Attestation{}
	}
	return &ethpb.Attestation{
		AggregationBits: v1alpha1Att.AggregationBits,
		Data:            V1Alpha1AttDataToV1(v1alpha1Att.Data),
		Signature:       v1alpha1Att.Signature,
	}
}

// V1AttestationToV1Alpha1 converts a v1 attestation to v1alpha1.
func V1AttestationToV1Alpha1(v1Att *ethpb.Attestation) *eth.Attestation {
	if v1Att == nil {
		return &eth.Attestation{}
	}
	return &eth.Attestation{
		AggregationBits: v1Att.AggregationBits,
		Data:            V1AttDataToV1Alpha1(v1Att.Data),
		Signature:       v1Att.Signature,
	}
}

// V1Alpha1AttDataToV1 converts a v1alpha1 attestation data to v1.
func V1Alpha1AttDataToV1(v1alpha1AttData *eth.AttestationData) *ethpb.AttestationData {
	if v1alpha1AttData == nil || v1alpha1AttData.Source == nil || v1alpha1AttData.Target == nil {
		return &ethpb.AttestationData{}
	}
	return &ethpb.AttestationData{
		Slot:            v1alpha1AttData.Slot,
		Index:           v1alpha1AttData.CommitteeIndex,
		BeaconBlockRoot: v1alpha1AttData.BeaconBlockRoot,
		Source: &ethpb.Checkpoint{
			Root:  v1alpha1AttData.Source.Root,
			Epoch: v1alpha1AttData.Source.Epoch,
		},
		Target: &ethpb.Checkpoint{
			Root:  v1alpha1AttData.Target.Root,
			Epoch: v1alpha1AttData.Target.Epoch,
		},
	}
}

// V1Alpha1AttSlashingToV1 converts a v1alpha1 attester slashing to v1.
func V1Alpha1AttSlashingToV1(v1alpha1Slashing *eth.AttesterSlashing) *ethpb.AttesterSlashing {
	if v1alpha1Slashing == nil {
		return &ethpb.AttesterSlashing{}
	}
	return &ethpb.AttesterSlashing{
		Attestation_1: V1Alpha1IndexedAttToV1(v1alpha1Slashing.Attestation_1),
		Attestation_2: V1Alpha1IndexedAttToV1(v1alpha1Slashing.Attestation_2),
	}
}

// V1Alpha1SignedHeaderToV1 converts a v1alpha1 signed beacon block header to v1.
func V1Alpha1SignedHeaderToV1(v1alpha1Hdr *eth.SignedBeaconBlockHeader) *ethpb.SignedBeaconBlockHeader {
	if v1alpha1Hdr == nil || v1alpha1Hdr.Header == nil {
		return &ethpb.SignedBeaconBlockHeader{}
	}
	return &ethpb.SignedBeaconBlockHeader{
		Message: &ethpb.BeaconBlockHeader{
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
func V1SignedHeaderToV1Alpha1(v1Header *ethpb.SignedBeaconBlockHeader) *eth.SignedBeaconBlockHeader {
	if v1Header == nil || v1Header.Message == nil {
		return &eth.SignedBeaconBlockHeader{}
	}
	return &eth.SignedBeaconBlockHeader{
		Header: &eth.BeaconBlockHeader{
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
func V1Alpha1ProposerSlashingToV1(v1alpha1Slashing *eth.ProposerSlashing) *ethpb.ProposerSlashing {
	if v1alpha1Slashing == nil {
		return &ethpb.ProposerSlashing{}
	}
	return &ethpb.ProposerSlashing{
		SignedHeader_1: V1Alpha1SignedHeaderToV1(v1alpha1Slashing.Header_1),
		SignedHeader_2: V1Alpha1SignedHeaderToV1(v1alpha1Slashing.Header_2),
	}
}

// V1Alpha1ExitToV1 converts a v1alpha1 SignedVoluntaryExit to v1.
func V1Alpha1ExitToV1(v1alpha1Exit *eth.SignedVoluntaryExit) *ethpb.SignedVoluntaryExit {
	if v1alpha1Exit == nil || v1alpha1Exit.Exit == nil {
		return &ethpb.SignedVoluntaryExit{}
	}
	return &ethpb.SignedVoluntaryExit{
		Message: &ethpb.VoluntaryExit{
			Epoch:          v1alpha1Exit.Exit.Epoch,
			ValidatorIndex: v1alpha1Exit.Exit.ValidatorIndex,
		},
		Signature: v1alpha1Exit.Signature,
	}
}

// V1ExitToV1Alpha1 converts a v1 SignedVoluntaryExit to v1alpha1.
func V1ExitToV1Alpha1(v1Exit *ethpb.SignedVoluntaryExit) *eth.SignedVoluntaryExit {
	if v1Exit == nil || v1Exit.Message == nil {
		return &eth.SignedVoluntaryExit{}
	}
	return &eth.SignedVoluntaryExit{
		Exit: &eth.VoluntaryExit{
			Epoch:          v1Exit.Message.Epoch,
			ValidatorIndex: v1Exit.Message.ValidatorIndex,
		},
		Signature: v1Exit.Signature,
	}
}

// V1AttToV1Alpha1 converts a v1 attestation to v1alpha1.
func V1AttToV1Alpha1(v1Att *ethpb.Attestation) *eth.Attestation {
	if v1Att == nil {
		return &eth.Attestation{}
	}
	return &eth.Attestation{
		AggregationBits: v1Att.AggregationBits,
		Data:            V1AttDataToV1Alpha1(v1Att.Data),
		Signature:       v1Att.Signature,
	}
}

// V1IndexedAttToV1Alpha1 converts a v1 indexed attestation to v1alpha1.
func V1IndexedAttToV1Alpha1(v1Att *ethpb.IndexedAttestation) *eth.IndexedAttestation {
	if v1Att == nil {
		return &eth.IndexedAttestation{}
	}
	return &eth.IndexedAttestation{
		AttestingIndices: v1Att.AttestingIndices,
		Data:             V1AttDataToV1Alpha1(v1Att.Data),
		Signature:        v1Att.Signature,
	}
}

// V1AttDataToV1Alpha1 converts a v1 attestation data to v1alpha1.
func V1AttDataToV1Alpha1(v1AttData *ethpb.AttestationData) *eth.AttestationData {
	if v1AttData == nil || v1AttData.Source == nil || v1AttData.Target == nil {
		return &eth.AttestationData{}
	}
	return &eth.AttestationData{
		Slot:            v1AttData.Slot,
		CommitteeIndex:  v1AttData.Index,
		BeaconBlockRoot: v1AttData.BeaconBlockRoot,
		Source: &eth.Checkpoint{
			Root:  v1AttData.Source.Root,
			Epoch: v1AttData.Source.Epoch,
		},
		Target: &eth.Checkpoint{
			Root:  v1AttData.Target.Root,
			Epoch: v1AttData.Target.Epoch,
		},
	}
}

// V1AttSlashingToV1Alpha1 converts a v1 attester slashing to v1alpha1.
func V1AttSlashingToV1Alpha1(v1Slashing *ethpb.AttesterSlashing) *eth.AttesterSlashing {
	if v1Slashing == nil {
		return &eth.AttesterSlashing{}
	}
	return &eth.AttesterSlashing{
		Attestation_1: V1IndexedAttToV1Alpha1(v1Slashing.Attestation_1),
		Attestation_2: V1IndexedAttToV1Alpha1(v1Slashing.Attestation_2),
	}
}

// V1ProposerSlashingToV1Alpha1 converts a v1 proposer slashing to v1alpha1.
func V1ProposerSlashingToV1Alpha1(v1Slashing *ethpb.ProposerSlashing) *eth.ProposerSlashing {
	if v1Slashing == nil {
		return &eth.ProposerSlashing{}
	}
	return &eth.ProposerSlashing{
		Header_1: V1SignedHeaderToV1Alpha1(v1Slashing.SignedHeader_1),
		Header_2: V1SignedHeaderToV1Alpha1(v1Slashing.SignedHeader_2),
	}
}

// V1Alpha1ValidatorToV1 converts a v1alpha1 validator to v1.
func V1Alpha1ValidatorToV1(v1Alpha1Validator *eth.Validator) *ethpb.Validator {
	if v1Alpha1Validator == nil {
		return &ethpb.Validator{}
	}
	return &ethpb.Validator{
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
func V1ValidatorToV1Alpha1(v1Validator *ethpb.Validator) *eth.Validator {
	if v1Validator == nil {
		return &eth.Validator{}
	}
	return &eth.Validator{
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
func SignedBeaconBlock(block block.SignedBeaconBlock) (*ethpb.SignedBeaconBlock, error) {
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
