//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c beaconApiValidatorClient) proposeBeaconBlock(in *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	var consensusVersion string
	var beaconBlockRoot []byte

	var err error
	var marshalledSignedBeaconBlockJson []byte
	blinded := false

	switch blockType := in.Block.(type) {
	case *ethpb.GenericSignedBeaconBlock_Phase0:
		consensusVersion = "phase0"
		if len(blockType.Phase0.Block.Body.Attestations) > 0 {
			beaconBlockRoot = blockType.Phase0.Block.Body.Attestations[0].Data.BeaconBlockRoot
		}

		marshalledSignedBeaconBlockJson, err = marshallBeaconBlockPhase0(blockType.Phase0)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshall phase0 beacon block")
		}
	case *ethpb.GenericSignedBeaconBlock_Altair:
		consensusVersion = "altair"
		if len(blockType.Altair.Block.Body.Attestations) > 0 {
			beaconBlockRoot = blockType.Altair.Block.Body.Attestations[0].Data.BeaconBlockRoot
		}

		marshalledSignedBeaconBlockJson, err = marshallBeaconBlockAltair(blockType.Altair)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshall altair beacon block")
		}
	case *ethpb.GenericSignedBeaconBlock_Bellatrix:
		consensusVersion = "bellatrix"
		if len(blockType.Bellatrix.Block.Body.Attestations) > 0 {
			beaconBlockRoot = blockType.Bellatrix.Block.Body.Attestations[0].Data.BeaconBlockRoot
		}

		marshalledSignedBeaconBlockJson, err = marshallBeaconBlockBellatrix(blockType.Bellatrix)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshall bellatrix beacon block")
		}
	case *ethpb.GenericSignedBeaconBlock_BlindedBellatrix:
		blinded = true
		consensusVersion = "bellatrix"
		if len(blockType.BlindedBellatrix.Block.Body.Attestations) > 0 {
			beaconBlockRoot = blockType.BlindedBellatrix.Block.Body.Attestations[0].Data.BeaconBlockRoot
		}

		marshalledSignedBeaconBlockJson, err = marshallBeaconBlockBlindedBellatrix(blockType.BlindedBellatrix)
		if err != nil {
			return nil, err
		}
	case *ethpb.GenericSignedBeaconBlock_BlindedCapella:
		blinded = true
		consensusVersion = "capella"
		if len(blockType.BlindedCapella.Block.Body.Attestations) > 0 {
			beaconBlockRoot = blockType.BlindedCapella.Block.Body.Attestations[0].Data.BeaconBlockRoot
		}

		marshalledSignedBeaconBlockJson, err = marshallBeaconBlockBlindedCapella(blockType.BlindedCapella)
		if err != nil {
			return nil, err
		}
	case *ethpb.GenericSignedBeaconBlock_Capella:
		return nil, errors.Errorf("Capella blocks are not supported yet")
	default:
		return nil, errors.Errorf("unsupported block type")
	}

	var endpoint string

	if blinded {
		endpoint = "/eth/v1/beacon/blinded_blocks"
	} else {
		endpoint = "/eth/v1/beacon/blocks"
	}

	headers := map[string]string{"Eth-Consensus-Version": consensusVersion}
	if _, err := c.jsonRestHandler.PostRestJson(endpoint, headers, bytes.NewBuffer(marshalledSignedBeaconBlockJson), nil); err != nil {
		return nil, errors.Wrap(err, "failed to send POST data to REST endpoint")
	}

	return &ethpb.ProposeResponse{BlockRoot: beaconBlockRoot}, nil
}

func marshallBeaconBlockPhase0(block *ethpb.SignedBeaconBlock) ([]byte, error) {
	signedBeaconBlockJson := &apimiddleware.SignedBeaconBlockContainerJson{
		Signature: hexutil.Encode(block.Signature),
		Message: &apimiddleware.BeaconBlockJson{
			Body:          jsonifyBeaconBlockBody(block.Block.Body),
			ParentRoot:    hexutil.Encode(block.Block.ParentRoot),
			ProposerIndex: uint64ToString(block.Block.ProposerIndex),
			Slot:          uint64ToString(block.Block.Slot),
			StateRoot:     hexutil.Encode(block.Block.StateRoot),
		},
	}

	return json.Marshal(signedBeaconBlockJson)
}

func marshallBeaconBlockAltair(block *ethpb.SignedBeaconBlockAltair) ([]byte, error) {
	// Convert the phase0 fields of Altair to a BeaconBlockBody to be able to reuse jsonifyBeaconBlockBody
	phase0BeaconBlockBodyJson := jsonifyBeaconBlockBody(block.Block.Body)

	signedBeaconBlockAltairJson := &apimiddleware.SignedBeaconBlockAltairContainerJson{
		Signature: hexutil.Encode(block.Signature),
		Message: &apimiddleware.BeaconBlockAltairJson{
			ParentRoot:    hexutil.Encode(block.Block.ParentRoot),
			ProposerIndex: uint64ToString(block.Block.ProposerIndex),
			Slot:          uint64ToString(block.Block.Slot),
			StateRoot:     hexutil.Encode(block.Block.StateRoot),
			Body: &apimiddleware.BeaconBlockBodyAltairJson{
				// Set the phase0 fields
				Attestations:      phase0BeaconBlockBodyJson.Attestations,
				AttesterSlashings: phase0BeaconBlockBodyJson.AttesterSlashings,
				Deposits:          phase0BeaconBlockBodyJson.Deposits,
				Eth1Data:          phase0BeaconBlockBodyJson.Eth1Data,
				Graffiti:          phase0BeaconBlockBodyJson.Graffiti,
				ProposerSlashings: phase0BeaconBlockBodyJson.ProposerSlashings,
				RandaoReveal:      phase0BeaconBlockBodyJson.RandaoReveal,
				VoluntaryExits:    phase0BeaconBlockBodyJson.VoluntaryExits,
				// Set the altair fields
				SyncAggregate: &apimiddleware.SyncAggregateJson{
					SyncCommitteeBits:      hexutil.Encode(block.Block.Body.SyncAggregate.SyncCommitteeBits),
					SyncCommitteeSignature: hexutil.Encode(block.Block.Body.SyncAggregate.SyncCommitteeSignature),
				},
			},
		},
	}

	return json.Marshal(signedBeaconBlockAltairJson)
}

func marshallBeaconBlockBellatrix(block *ethpb.SignedBeaconBlockBellatrix) ([]byte, error) {
	// Gather the transactions
	var executionPayloadTransaction []string
	for _, transaction := range block.Block.Body.ExecutionPayload.Transactions {
		transactionJson := hexutil.Encode(transaction)
		executionPayloadTransaction = append(executionPayloadTransaction, transactionJson)
	}

	// Convert the phase0 fields of Bellatrix to a BeaconBlockBody to be able to reuse jsonifyBeaconBlockBody
	phase0BeaconBlockBodyJson := jsonifyBeaconBlockBody(block.Block.Body)

	signedBeaconBlockBellatrixJson := &apimiddleware.SignedBeaconBlockBellatrixContainerJson{
		Signature: hexutil.Encode(block.Signature),
		Message: &apimiddleware.BeaconBlockBellatrixJson{
			ParentRoot:    hexutil.Encode(block.Block.ParentRoot),
			ProposerIndex: uint64ToString(block.Block.ProposerIndex),
			Slot:          uint64ToString(block.Block.Slot),
			StateRoot:     hexutil.Encode(block.Block.StateRoot),
			Body: &apimiddleware.BeaconBlockBodyBellatrixJson{
				// Set the phase0 fields
				Attestations:      phase0BeaconBlockBodyJson.Attestations,
				AttesterSlashings: phase0BeaconBlockBodyJson.AttesterSlashings,
				Deposits:          phase0BeaconBlockBodyJson.Deposits,
				Eth1Data:          phase0BeaconBlockBodyJson.Eth1Data,
				Graffiti:          phase0BeaconBlockBodyJson.Graffiti,
				ProposerSlashings: phase0BeaconBlockBodyJson.ProposerSlashings,
				RandaoReveal:      phase0BeaconBlockBodyJson.RandaoReveal,
				VoluntaryExits:    phase0BeaconBlockBodyJson.VoluntaryExits,
				// Set the altair fields
				SyncAggregate: &apimiddleware.SyncAggregateJson{
					SyncCommitteeBits:      hexutil.Encode(block.Block.Body.SyncAggregate.SyncCommitteeBits),
					SyncCommitteeSignature: hexutil.Encode(block.Block.Body.SyncAggregate.SyncCommitteeSignature),
				},
				// Set the bellatrix fields
				ExecutionPayload: &apimiddleware.ExecutionPayloadJson{
					BaseFeePerGas: uint256BytesToString(block.Block.Body.ExecutionPayload.BaseFeePerGas),
					BlockHash:     hexutil.Encode(block.Block.Body.ExecutionPayload.BlockHash),
					BlockNumber:   uint64ToString(block.Block.Body.ExecutionPayload.BlockNumber),
					ExtraData:     hexutil.Encode(block.Block.Body.ExecutionPayload.ExtraData),
					FeeRecipient:  hexutil.Encode(block.Block.Body.ExecutionPayload.FeeRecipient),
					GasLimit:      uint64ToString(block.Block.Body.ExecutionPayload.GasLimit),
					GasUsed:       uint64ToString(block.Block.Body.ExecutionPayload.GasUsed),
					LogsBloom:     hexutil.Encode(block.Block.Body.ExecutionPayload.LogsBloom),
					ParentHash:    hexutil.Encode(block.Block.Body.ExecutionPayload.ParentHash),
					PrevRandao:    hexutil.Encode(block.Block.Body.ExecutionPayload.PrevRandao),
					ReceiptsRoot:  hexutil.Encode(block.Block.Body.ExecutionPayload.ReceiptsRoot),
					StateRoot:     hexutil.Encode(block.Block.Body.ExecutionPayload.StateRoot),
					TimeStamp:     uint64ToString(block.Block.Body.ExecutionPayload.Timestamp),
					Transactions:  executionPayloadTransaction,
				},
			},
		},
	}

	return json.Marshal(signedBeaconBlockBellatrixJson)
}

func marshallBeaconBlockBlindedBellatrix(block *ethpb.SignedBlindedBeaconBlockBellatrix) ([]byte, error) {
	// Convert the phase0 fields of BlindedBellatrix to a BeaconBlockBody to be able to reuse jsonifyBeaconBlockBody
	phase0BeaconBlockBodyJson := jsonifyBeaconBlockBody(block.Block.Body)

	signedBeaconBlockBellatrixJson := &apimiddleware.SignedBlindedBeaconBlockBellatrixContainerJson{
		Signature: hexutil.Encode(block.Signature),
		Message: &apimiddleware.BlindedBeaconBlockBellatrixJson{
			ParentRoot:    hexutil.Encode(block.Block.ParentRoot),
			ProposerIndex: uint64ToString(block.Block.ProposerIndex),
			Slot:          uint64ToString(block.Block.Slot),
			StateRoot:     hexutil.Encode(block.Block.StateRoot),
			Body: &apimiddleware.BlindedBeaconBlockBodyBellatrixJson{
				// Set the phase0 fields
				Attestations:      phase0BeaconBlockBodyJson.Attestations,
				AttesterSlashings: phase0BeaconBlockBodyJson.AttesterSlashings,
				Deposits:          phase0BeaconBlockBodyJson.Deposits,
				Eth1Data:          phase0BeaconBlockBodyJson.Eth1Data,
				Graffiti:          phase0BeaconBlockBodyJson.Graffiti,
				ProposerSlashings: phase0BeaconBlockBodyJson.ProposerSlashings,
				RandaoReveal:      phase0BeaconBlockBodyJson.RandaoReveal,
				VoluntaryExits:    phase0BeaconBlockBodyJson.VoluntaryExits,
				// Set the altair fields
				SyncAggregate: &apimiddleware.SyncAggregateJson{
					SyncCommitteeBits:      hexutil.Encode(block.Block.Body.SyncAggregate.SyncCommitteeBits),
					SyncCommitteeSignature: hexutil.Encode(block.Block.Body.SyncAggregate.SyncCommitteeSignature),
				},
				// Set the bellatrix fields
				ExecutionPayloadHeader: &apimiddleware.ExecutionPayloadHeaderJson{
					BaseFeePerGas:    uint256BytesToString(block.Block.Body.ExecutionPayloadHeader.BaseFeePerGas),
					BlockHash:        hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.BlockHash),
					BlockNumber:      uint64ToString(block.Block.Body.ExecutionPayloadHeader.BlockNumber),
					ExtraData:        hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.ExtraData),
					FeeRecipient:     hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.FeeRecipient),
					GasLimit:         uint64ToString(block.Block.Body.ExecutionPayloadHeader.GasLimit),
					GasUsed:          uint64ToString(block.Block.Body.ExecutionPayloadHeader.GasUsed),
					LogsBloom:        hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.LogsBloom),
					ParentHash:       hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.ParentHash),
					PrevRandao:       hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.PrevRandao),
					ReceiptsRoot:     hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.ReceiptsRoot),
					StateRoot:        hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.StateRoot),
					TimeStamp:        uint64ToString(block.Block.Body.ExecutionPayloadHeader.Timestamp),
					TransactionsRoot: hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.TransactionsRoot),
				},
			},
		},
	}

	return json.Marshal(signedBeaconBlockBellatrixJson)
}

func marshallBeaconBlockBlindedCapella(block *ethpb.SignedBlindedBeaconBlockCapella) ([]byte, error) {
	phase0BeaconBlockBodyJson := jsonifyBeaconBlockBody(block.Block.Body)

	blsToExecutionChanges := make([]*apimiddleware.BLSToExecutionChangeJson, 0, len(block.Block.Body.BlsToExecutionChanges))

	for _, signedBlsToExecutionChange := range block.Block.Body.BlsToExecutionChanges {
		blsToExecutionChangeJson := &apimiddleware.BLSToExecutionChangeJson{
			ValidatorIndex:     uint64ToString(signedBlsToExecutionChange.Message.ValidatorIndex),
			FromBLSPubkey:      hexutil.Encode(signedBlsToExecutionChange.Message.FromBlsPubkey),
			ToExecutionAddress: hexutil.Encode(signedBlsToExecutionChange.Message.ToExecutionAddress),
		}
		blsToExecutionChanges = append(blsToExecutionChanges, blsToExecutionChangeJson)
	}

	signedBeaconBlockCapellaJson := &apimiddleware.SignedBlindedBeaconBlockCapellaContainerJson{
		Signature: hexutil.Encode(block.Signature),
		Message: &apimiddleware.BlindedBeaconBlockCapellaJson{
			ParentRoot:    hexutil.Encode(block.Block.ParentRoot),
			ProposerIndex: uint64ToString(block.Block.ProposerIndex),
			Slot:          uint64ToString(block.Block.Slot),
			StateRoot:     hexutil.Encode(block.Block.StateRoot),
			Body: &apimiddleware.BlindedBeaconBlockBodyCapellaJson{
				// Set the phase0 fields
				Attestations:      phase0BeaconBlockBodyJson.Attestations,
				AttesterSlashings: phase0BeaconBlockBodyJson.AttesterSlashings,
				Deposits:          phase0BeaconBlockBodyJson.Deposits,
				Eth1Data:          phase0BeaconBlockBodyJson.Eth1Data,
				Graffiti:          phase0BeaconBlockBodyJson.Graffiti,
				ProposerSlashings: phase0BeaconBlockBodyJson.ProposerSlashings,
				RandaoReveal:      phase0BeaconBlockBodyJson.RandaoReveal,
				VoluntaryExits:    phase0BeaconBlockBodyJson.VoluntaryExits,
				// Set the altair fields
				SyncAggregate: &apimiddleware.SyncAggregateJson{
					SyncCommitteeBits:      hexutil.Encode(block.Block.Body.SyncAggregate.SyncCommitteeBits),
					SyncCommitteeSignature: hexutil.Encode(block.Block.Body.SyncAggregate.SyncCommitteeSignature),
				},
				BLSToExecutionChanges: blsToExecutionChanges,
				// Set the capella fields
				ExecutionPayloadHeader: &apimiddleware.ExecutionPayloadHeaderCapellaJson{
					BaseFeePerGas:    uint256BytesToString(block.Block.Body.ExecutionPayloadHeader.BaseFeePerGas),
					BlockHash:        hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.BlockHash),
					BlockNumber:      uint64ToString(block.Block.Body.ExecutionPayloadHeader.BlockNumber),
					ExtraData:        hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.ExtraData),
					FeeRecipient:     hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.FeeRecipient),
					GasLimit:         uint64ToString(block.Block.Body.ExecutionPayloadHeader.GasLimit),
					GasUsed:          uint64ToString(block.Block.Body.ExecutionPayloadHeader.GasUsed),
					LogsBloom:        hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.LogsBloom),
					ParentHash:       hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.ParentHash),
					PrevRandao:       hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.PrevRandao),
					ReceiptsRoot:     hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.ReceiptsRoot),
					StateRoot:        hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.StateRoot),
					TimeStamp:        uint64ToString(block.Block.Body.ExecutionPayloadHeader.Timestamp),
					TransactionsRoot: hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.TransactionsRoot),
					WithdrawalsRoot:  hexutil.Encode(block.Block.Body.ExecutionPayloadHeader.WithdrawalsRoot),
				},
			},
		},
	}

	return json.Marshal(signedBeaconBlockCapellaJson)
}

type phase0BeaconBlockBody interface {
	GetRandaoReveal() []byte
	GetEth1Data() *ethpb.Eth1Data
	GetGraffiti() []byte
	GetProposerSlashings() []*ethpb.ProposerSlashing
	GetAttesterSlashings() []*ethpb.AttesterSlashing
	GetAttestations() []*ethpb.Attestation
	GetDeposits() []*ethpb.Deposit
	GetVoluntaryExits() []*ethpb.SignedVoluntaryExit
}

func jsonifyBeaconBlockBody(beaconBlockBody phase0BeaconBlockBody) *apimiddleware.BeaconBlockBodyJson {
	attestations := []*apimiddleware.AttestationJson{}
	for _, attestation := range beaconBlockBody.GetAttestations() {
		attestationJson := &apimiddleware.AttestationJson{
			AggregationBits: hexutil.Encode(attestation.AggregationBits),
			Data:            jsonifyAttestationData(attestation.Data),
			Signature:       hexutil.Encode(attestation.Signature),
		}
		attestations = append(attestations, attestationJson)
	}

	attesterSlashings := []*apimiddleware.AttesterSlashingJson{}
	for _, attesterSlashing := range beaconBlockBody.GetAttesterSlashings() {
		attesterSlashingJson := &apimiddleware.AttesterSlashingJson{
			Attestation_1: jsonifyIndexedAttestation(attesterSlashing.Attestation_1),
			Attestation_2: jsonifyIndexedAttestation(attesterSlashing.Attestation_2),
		}
		attesterSlashings = append(attesterSlashings, attesterSlashingJson)
	}

	deposits := []*apimiddleware.DepositJson{}
	for _, deposit := range beaconBlockBody.GetDeposits() {
		var proofs []string
		for _, proof := range deposit.Proof {
			proofs = append(proofs, hexutil.Encode(proof))
		}

		depositJson := &apimiddleware.DepositJson{
			Data: &apimiddleware.Deposit_DataJson{
				Amount:                uint64ToString(deposit.Data.Amount),
				PublicKey:             hexutil.Encode(deposit.Data.PublicKey),
				Signature:             hexutil.Encode(deposit.Data.Signature),
				WithdrawalCredentials: hexutil.Encode(deposit.Data.WithdrawalCredentials),
			},
			Proof: proofs,
		}
		deposits = append(deposits, depositJson)
	}

	proposerSlashings := []*apimiddleware.ProposerSlashingJson{}
	for _, proposerSlashing := range beaconBlockBody.GetProposerSlashings() {
		proposerSlashingJson := &apimiddleware.ProposerSlashingJson{
			Header_1: jsonifySignedBeaconBlockHeader(proposerSlashing.Header_1),
			Header_2: jsonifySignedBeaconBlockHeader(proposerSlashing.Header_2),
		}
		proposerSlashings = append(proposerSlashings, proposerSlashingJson)
	}

	signedVoluntaryExits := []*apimiddleware.SignedVoluntaryExitJson{}
	for _, signedVoluntaryExit := range beaconBlockBody.GetVoluntaryExits() {
		signedVoluntaryExitJson := &apimiddleware.SignedVoluntaryExitJson{
			Exit: &apimiddleware.VoluntaryExitJson{
				Epoch:          uint64ToString(signedVoluntaryExit.Exit.Epoch),
				ValidatorIndex: uint64ToString(signedVoluntaryExit.Exit.ValidatorIndex),
			},
			Signature: hexutil.Encode(signedVoluntaryExit.Signature),
		}
		signedVoluntaryExits = append(signedVoluntaryExits, signedVoluntaryExitJson)
	}

	beaconBlockBodyJson := &apimiddleware.BeaconBlockBodyJson{
		Attestations:      attestations,
		AttesterSlashings: attesterSlashings,
		Deposits:          deposits,
		Eth1Data: &apimiddleware.Eth1DataJson{
			BlockHash:    hexutil.Encode(beaconBlockBody.GetEth1Data().BlockHash),
			DepositCount: uint64ToString(beaconBlockBody.GetEth1Data().DepositCount),
			DepositRoot:  hexutil.Encode(beaconBlockBody.GetEth1Data().DepositRoot),
		},
		Graffiti:          hexutil.Encode(beaconBlockBody.GetGraffiti()),
		ProposerSlashings: proposerSlashings,
		RandaoReveal:      hexutil.Encode(beaconBlockBody.GetRandaoReveal()),
		VoluntaryExits:    signedVoluntaryExits,
	}

	return beaconBlockBodyJson
}

func jsonifySignedBeaconBlockHeader(signedBeaconBlockHeader *ethpb.SignedBeaconBlockHeader) *apimiddleware.SignedBeaconBlockHeaderJson {
	return &apimiddleware.SignedBeaconBlockHeaderJson{
		Header: &apimiddleware.BeaconBlockHeaderJson{
			BodyRoot:      hexutil.Encode(signedBeaconBlockHeader.Header.BodyRoot),
			ParentRoot:    hexutil.Encode(signedBeaconBlockHeader.Header.ParentRoot),
			ProposerIndex: uint64ToString(signedBeaconBlockHeader.Header.ProposerIndex),
			Slot:          uint64ToString(signedBeaconBlockHeader.Header.Slot),
			StateRoot:     hexutil.Encode(signedBeaconBlockHeader.Header.StateRoot),
		},
		Signature: hexutil.Encode(signedBeaconBlockHeader.Signature),
	}
}

func jsonifyIndexedAttestation(indexedAttestation *ethpb.IndexedAttestation) *apimiddleware.IndexedAttestationJson {
	attestingIndices := make([]string, 0, len(indexedAttestation.AttestingIndices))
	for _, attestingIndex := range indexedAttestation.AttestingIndices {
		attestingIndex := uint64ToString(attestingIndex)
		attestingIndices = append(attestingIndices, attestingIndex)
	}

	return &apimiddleware.IndexedAttestationJson{
		Data:      jsonifyAttestationData(indexedAttestation.Data),
		Signature: hexutil.Encode(indexedAttestation.Signature),
	}
}

func jsonifyAttestationData(attestationData *ethpb.AttestationData) *apimiddleware.AttestationDataJson {
	return &apimiddleware.AttestationDataJson{
		BeaconBlockRoot: hexutil.Encode(attestationData.BeaconBlockRoot),
		CommitteeIndex:  uint64ToString(attestationData.CommitteeIndex),
		Slot:            uint64ToString(attestationData.Slot),
		Source: &apimiddleware.CheckpointJson{
			Epoch: uint64ToString(attestationData.Source.Epoch),
			Root:  hexutil.Encode(attestationData.Source.Root),
		},
		Target: &apimiddleware.CheckpointJson{
			Epoch: uint64ToString(attestationData.Target.Epoch),
			Root:  hexutil.Encode(attestationData.Target.Root),
		},
	}
}

func uint256BytesToString(bytes []byte) string {
	// Integers are stored as little-endian, but big.Int expects big-endian. So we need to reverse the byte order before decoding.
	return new(big.Int).SetBytes(bytesutil.ReverseByteOrder(bytes)).String()
}
