package v1

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
)

// GetBlockSignRequest maps the request for signing type BLOCK.
func GetBlockSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*BlockSignRequest, error) {
	beaconBlock := request.Object.(*validatorpb.SignRequest_Block)
	if beaconBlock == nil {
		return nil, errors.New("invalid sign request: BeaconBlock is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	beaconBlockBody, err := MapBeaconBlockBody(beaconBlock.Block.Body)
	if err != nil {
		return nil, err
	}
	return &BlockSignRequest{
		Type:        "BLOCK",
		ForkInfo:    fork,
		SigningRoot: hexutil.Encode(request.SigningRoot),
		Block: &BeaconBlock{
			Slot:          fmt.Sprint(beaconBlock.Block.Slot),
			ProposerIndex: fmt.Sprint(beaconBlock.Block.ProposerIndex),
			ParentRoot:    hexutil.Encode(beaconBlock.Block.ParentRoot),
			StateRoot:     hexutil.Encode(beaconBlock.Block.StateRoot),
			Body:          beaconBlockBody,
		},
	}, nil
}

// GetAggregationSlotSignRequest maps the request for signing type AGGREGATION_SLOT.
func GetAggregationSlotSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*AggregationSlotSignRequest, error) {
	aggregationSlot := request.Object.(*validatorpb.SignRequest_Slot)
	if aggregationSlot == nil {
		return nil, errors.New("invalid sign request: Slot is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	return &AggregationSlotSignRequest{
		Type:        "AGGREGATION_SLOT",
		ForkInfo:    fork,
		SigningRoot: hexutil.Encode(request.SigningRoot),
		AggregationSlot: &AggregationSlot{
			Slot: fmt.Sprint(aggregationSlot.Slot),
		},
	}, nil
}

// GetAggregateAndProofSignRequest maps the request for signing type AGGREGATE_AND_PROOF.
func GetAggregateAndProofSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*AggregateAndProofSignRequest, error) {
	aggregateAttestationAndProof := request.Object.(*validatorpb.SignRequest_AggregateAttestationAndProof)
	if aggregateAttestationAndProof == nil {
		return nil, errors.New("invalid sign request: AggregateAndProof is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	aggregateAndProof, err := MapAggregateAndProof(aggregateAttestationAndProof.AggregateAttestationAndProof)
	if err != nil {
		return nil, err
	}
	return &AggregateAndProofSignRequest{
		Type:              "AGGREGATE_AND_PROOF",
		ForkInfo:          fork,
		SigningRoot:       hexutil.Encode(request.SigningRoot),
		AggregateAndProof: aggregateAndProof,
	}, nil
}

// GetAttestationSignRequest maps the request for signing type ATTESTATION.
func GetAttestationSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*AttestationSignRequest, error) {
	attestation := request.Object.(*validatorpb.SignRequest_AttestationData)
	if attestation == nil {
		return nil, errors.New("invalid sign request: Attestation is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	attestationData, err := MapAttestationData(attestation.AttestationData)
	if err != nil {
		return nil, err
	}
	return &AttestationSignRequest{
		Type:        "ATTESTATION",
		ForkInfo:    fork,
		SigningRoot: hexutil.Encode(request.SigningRoot),
		Attestation: attestationData,
	}, nil
}

// GetBlockV2AltairSignRequest maps the request for signing type BLOCK_V2.
func GetBlockV2AltairSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*BlockV2AltairSignRequest, error) {
	beaconBlockV2 := request.Object.(*validatorpb.SignRequest_BlockV2)
	if beaconBlockV2 == nil {
		return nil, errors.New("invalid sign request: BeaconBlock is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	beaconBlockAltair, err := MapBeaconBlockAltair(beaconBlockV2.BlockV2)
	if err != nil {
		return nil, err
	}
	return &BlockV2AltairSignRequest{
		Type:        "BLOCK_V2",
		ForkInfo:    fork,
		SigningRoot: hexutil.Encode(request.SigningRoot),
		BeaconBlock: &BeaconBlockAltairBlockV2{
			Version: "ALTAIR",
			Block:   beaconBlockAltair,
		},
	}, nil
}

// GetRandaoRevealSignRequest maps the request for signing type RANDAO_REVEAL.
func GetRandaoRevealSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*RandaoRevealSignRequest, error) {
	randaoReveal := request.Object.(*validatorpb.SignRequest_Epoch)
	if randaoReveal == nil {
		return nil, errors.New("invalid sign request: Epoch is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	return &RandaoRevealSignRequest{
		Type:        "RANDAO_REVEAL",
		ForkInfo:    fork,
		SigningRoot: hexutil.Encode(request.SigningRoot),
		RandaoReveal: &RandaoReveal{
			Epoch: fmt.Sprint(randaoReveal.Epoch),
		},
	}, nil
}

// GetVoluntaryExitSignRequest maps the request for signing type VOLUNTARY_EXIT.
func GetVoluntaryExitSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*VoluntaryExitSignRequest, error) {
	voluntaryExit := request.Object.(*validatorpb.SignRequest_Exit).Exit
	if voluntaryExit == nil {
		return nil, errors.New("invalid sign request: Exit is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	return &VoluntaryExitSignRequest{
		Type:        "VOLUNTARY_EXIT",
		ForkInfo:    fork,
		SigningRoot: hexutil.Encode(request.SigningRoot),
		VoluntaryExit: &VoluntaryExit{
			ValidatorIndex: fmt.Sprint(voluntaryExit.ValidatorIndex),
			Epoch:          fmt.Sprint(voluntaryExit.Epoch),
		},
	}, nil
}

// GetSyncCommitteeMessageSignRequest maps the request for signing type SYNC_COMMITTEE_MESSAGE.
func GetSyncCommitteeMessageSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*SyncCommitteeMessageSignRequest, error) {
	syncCommitteeMessage := request.Object.(*validatorpb.SignRequest_SyncMessageBlockRoot)
	if syncCommitteeMessage == nil || syncCommitteeMessage.SyncMessageBlockRoot == nil {
		return nil, errors.New("invalid sign request: SyncCommitteeMessage is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	return &SyncCommitteeMessageSignRequest{
		Type:        "SYNC_COMMITTEE_MESSAGE",
		ForkInfo:    fork,
		SigningRoot: hexutil.Encode(request.SigningRoot),
		SyncCommitteeMessage: &SyncCommitteeMessage{
			BeaconBlockRoot: hexutil.Encode(syncCommitteeMessage.SyncMessageBlockRoot),
			Slot:            fmt.Sprint(request.SigningSlot),
		},
	}, nil
}

// GetSyncCommitteeSelectionProofSignRequest maps the request for signing type SYNC_COMMITTEE_SELECTION_PROOF.
func GetSyncCommitteeSelectionProofSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*SyncCommitteeSelectionProofSignRequest, error) {
	syncCommitteeSelectionProof := request.Object.(*validatorpb.SignRequest_SyncAggregatorSelectionData)
	if syncCommitteeSelectionProof == nil {
		return nil, errors.New("invalid sign request: SyncCommitteeSelectionProof is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	aggregatorSelectionData, err := MapSyncAggregatorSelectionData(syncCommitteeSelectionProof.SyncAggregatorSelectionData)
	if err != nil {
		return nil, err
	}
	return &SyncCommitteeSelectionProofSignRequest{
		Type:                        "SYNC_COMMITTEE_SELECTION_PROOF",
		ForkInfo:                    fork,
		SigningRoot:                 hexutil.Encode(request.SigningRoot),
		SyncAggregatorSelectionData: aggregatorSelectionData,
	}, nil
}

// GetSyncCommitteeContributionAndProofSignRequest maps the request for signing type SYNC_COMMITTEE_CONTRIBUTION_AND_PROOF.
func GetSyncCommitteeContributionAndProofSignRequest(request *validatorpb.SignRequest, genesisValidatorsRoot []byte) (*SyncCommitteeContributionAndProofSignRequest, error) {
	syncCommitteeContributionAndProof := request.Object.(*validatorpb.SignRequest_ContributionAndProof)
	if syncCommitteeContributionAndProof == nil {
		return nil, errors.New("invalid sign request: SyncCommitteeContributionAndProof is nil")
	}
	fork, err := MapForkInfo(request.SigningSlot, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	contribution, err := MapContributionAndProof(syncCommitteeContributionAndProof.ContributionAndProof)
	if err != nil {
		return nil, err
	}
	return &SyncCommitteeContributionAndProofSignRequest{
		Type:                 "SYNC_COMMITTEE_CONTRIBUTION_AND_PROOF",
		ForkInfo:             fork,
		SigningRoot:          hexutil.Encode(request.SigningRoot),
		ContributionAndProof: contribution,
	}, nil
}
