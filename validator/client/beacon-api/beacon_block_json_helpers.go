package beacon_api

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/apimiddleware"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

func jsonifyTransactions(transactions [][]byte) []string {
	jsonTransactions := make([]string, len(transactions))
	for index, transaction := range transactions {
		jsonTransaction := hexutil.Encode(transaction)
		jsonTransactions[index] = jsonTransaction
	}
	return jsonTransactions
}

// TODO: remove when apimiddleware is migrated away
func jsonifyBlsToExecutionChanges(blsToExecutionChanges []*ethpb.SignedBLSToExecutionChange) []*apimiddleware.SignedBLSToExecutionChangeJson {
	jsonBlsToExecutionChanges := make([]*apimiddleware.SignedBLSToExecutionChangeJson, len(blsToExecutionChanges))
	for index, signedBlsToExecutionChange := range blsToExecutionChanges {
		blsToExecutionChangeJson := &apimiddleware.BLSToExecutionChangeJson{
			ValidatorIndex:     uint64ToString(signedBlsToExecutionChange.Message.ValidatorIndex),
			FromBLSPubkey:      hexutil.Encode(signedBlsToExecutionChange.Message.FromBlsPubkey),
			ToExecutionAddress: hexutil.Encode(signedBlsToExecutionChange.Message.ToExecutionAddress),
		}
		signedJson := &apimiddleware.SignedBLSToExecutionChangeJson{
			Message:   blsToExecutionChangeJson,
			Signature: hexutil.Encode(signedBlsToExecutionChange.Signature),
		}
		jsonBlsToExecutionChanges[index] = signedJson
	}
	return jsonBlsToExecutionChanges
}

func jsonifyEth1Data(eth1Data *ethpb.Eth1Data) *apimiddleware.Eth1DataJson {
	return &apimiddleware.Eth1DataJson{
		BlockHash:    hexutil.Encode(eth1Data.BlockHash),
		DepositCount: uint64ToString(eth1Data.DepositCount),
		DepositRoot:  hexutil.Encode(eth1Data.DepositRoot),
	}
}

func jsonifyAttestations(attestations []*ethpb.Attestation) []*apimiddleware.AttestationJson {
	jsonAttestations := make([]*apimiddleware.AttestationJson, len(attestations))
	for index, attestation := range attestations {
		jsonAttestations[index] = jsonifyAttestation(attestation)
	}
	return jsonAttestations
}

func jsonifyAttesterSlashings(attesterSlashings []*ethpb.AttesterSlashing) []*apimiddleware.AttesterSlashingJson {
	jsonAttesterSlashings := make([]*apimiddleware.AttesterSlashingJson, len(attesterSlashings))
	for index, attesterSlashing := range attesterSlashings {
		jsonAttesterSlashing := &apimiddleware.AttesterSlashingJson{
			Attestation_1: jsonifyIndexedAttestation(attesterSlashing.Attestation_1),
			Attestation_2: jsonifyIndexedAttestation(attesterSlashing.Attestation_2),
		}
		jsonAttesterSlashings[index] = jsonAttesterSlashing
	}
	return jsonAttesterSlashings
}

func jsonifyDeposits(deposits []*ethpb.Deposit) []*apimiddleware.DepositJson {
	jsonDeposits := make([]*apimiddleware.DepositJson, len(deposits))
	for depositIndex, deposit := range deposits {
		proofs := make([]string, len(deposit.Proof))
		for proofIndex, proof := range deposit.Proof {
			proofs[proofIndex] = hexutil.Encode(proof)
		}

		jsonDeposit := &apimiddleware.DepositJson{
			Data: &apimiddleware.Deposit_DataJson{
				Amount:                uint64ToString(deposit.Data.Amount),
				PublicKey:             hexutil.Encode(deposit.Data.PublicKey),
				Signature:             hexutil.Encode(deposit.Data.Signature),
				WithdrawalCredentials: hexutil.Encode(deposit.Data.WithdrawalCredentials),
			},
			Proof: proofs,
		}
		jsonDeposits[depositIndex] = jsonDeposit
	}
	return jsonDeposits
}

func jsonifyProposerSlashings(proposerSlashings []*ethpb.ProposerSlashing) []*apimiddleware.ProposerSlashingJson {
	jsonProposerSlashings := make([]*apimiddleware.ProposerSlashingJson, len(proposerSlashings))
	for index, proposerSlashing := range proposerSlashings {
		jsonProposerSlashing := &apimiddleware.ProposerSlashingJson{
			Header_1: jsonifySignedBeaconBlockHeader(proposerSlashing.Header_1),
			Header_2: jsonifySignedBeaconBlockHeader(proposerSlashing.Header_2),
		}
		jsonProposerSlashings[index] = jsonProposerSlashing
	}
	return jsonProposerSlashings
}

// JsonifySignedVoluntaryExits converts an array of voluntary exit structs to a JSON hex string compatible format.
func JsonifySignedVoluntaryExits(voluntaryExits []*ethpb.SignedVoluntaryExit) []*apimiddleware.SignedVoluntaryExitJson {
	jsonSignedVoluntaryExits := make([]*apimiddleware.SignedVoluntaryExitJson, len(voluntaryExits))
	for index, signedVoluntaryExit := range voluntaryExits {
		jsonSignedVoluntaryExit := &apimiddleware.SignedVoluntaryExitJson{
			Exit: &apimiddleware.VoluntaryExitJson{
				Epoch:          uint64ToString(signedVoluntaryExit.Exit.Epoch),
				ValidatorIndex: uint64ToString(signedVoluntaryExit.Exit.ValidatorIndex),
			},
			Signature: hexutil.Encode(signedVoluntaryExit.Signature),
		}
		jsonSignedVoluntaryExits[index] = jsonSignedVoluntaryExit
	}
	return jsonSignedVoluntaryExits
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
	attestingIndices := make([]string, len(indexedAttestation.AttestingIndices))
	for index, attestingIndex := range indexedAttestation.AttestingIndices {
		attestingIndex := uint64ToString(attestingIndex)
		attestingIndices[index] = attestingIndex
	}

	return &apimiddleware.IndexedAttestationJson{
		AttestingIndices: attestingIndices,
		Data:             jsonifyAttestationData(indexedAttestation.Data),
		Signature:        hexutil.Encode(indexedAttestation.Signature),
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

func jsonifyAttestation(attestation *ethpb.Attestation) *apimiddleware.AttestationJson {
	return &apimiddleware.AttestationJson{
		AggregationBits: hexutil.Encode(attestation.AggregationBits),
		Data:            jsonifyAttestationData(attestation.Data),
		Signature:       hexutil.Encode(attestation.Signature),
	}
}

func jsonifySignedAggregateAndProof(signedAggregateAndProof *ethpb.SignedAggregateAttestationAndProof) *apimiddleware.SignedAggregateAttestationAndProofJson {
	return &apimiddleware.SignedAggregateAttestationAndProofJson{
		Message: &apimiddleware.AggregateAttestationAndProofJson{
			AggregatorIndex: uint64ToString(signedAggregateAndProof.Message.AggregatorIndex),
			Aggregate:       jsonifyAttestation(signedAggregateAndProof.Message.Aggregate),
			SelectionProof:  hexutil.Encode(signedAggregateAndProof.Message.SelectionProof),
		},
		Signature: hexutil.Encode(signedAggregateAndProof.Signature),
	}
}

func jsonifyWithdrawals(withdrawals []*enginev1.Withdrawal) []*apimiddleware.WithdrawalJson {
	jsonWithdrawals := make([]*apimiddleware.WithdrawalJson, len(withdrawals))
	for index, withdrawal := range withdrawals {
		jsonWithdrawals[index] = &apimiddleware.WithdrawalJson{
			WithdrawalIndex:  strconv.FormatUint(withdrawal.Index, 10),
			ValidatorIndex:   strconv.FormatUint(uint64(withdrawal.ValidatorIndex), 10),
			ExecutionAddress: hexutil.Encode(withdrawal.Address),
			Amount:           strconv.FormatUint(withdrawal.Amount, 10),
		}
	}
	return jsonWithdrawals
}
