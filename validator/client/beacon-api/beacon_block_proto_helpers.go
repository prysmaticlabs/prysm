package beacon_api

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func convertProposerSlashingsToProto(jsonProposerSlashings []*apimiddleware.ProposerSlashingJson) ([]*ethpb.ProposerSlashing, error) {
	proposerSlashings := make([]*ethpb.ProposerSlashing, len(jsonProposerSlashings))

	for index, jsonProposerSlashing := range jsonProposerSlashings {
		if jsonProposerSlashing == nil {
			return nil, errors.Errorf("proposer slashing at index `%d` is nil", index)
		}

		header1, err := convertProposerSlashingSignedHeaderToProto(jsonProposerSlashing.Header_1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get proposer header 1")
		}

		header2, err := convertProposerSlashingSignedHeaderToProto(jsonProposerSlashing.Header_2)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get proposer header 2")
		}

		proposerSlashings[index] = &ethpb.ProposerSlashing{
			Header_1: header1,
			Header_2: header2,
		}
	}

	return proposerSlashings, nil
}

func convertProposerSlashingSignedHeaderToProto(signedHeader *apimiddleware.SignedBeaconBlockHeaderJson) (*ethpb.SignedBeaconBlockHeader, error) {
	if signedHeader == nil {
		return nil, errors.New("signed header is nil")
	}

	if signedHeader.Header == nil {
		return nil, errors.New("header is nil")
	}

	slot, err := strconv.ParseUint(signedHeader.Header.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse header slot `%s`", signedHeader.Header.Slot)
	}

	proposerIndex, err := strconv.ParseUint(signedHeader.Header.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse header proposer index `%s`", signedHeader.Header.ProposerIndex)
	}

	parentRoot, err := hexutil.Decode(signedHeader.Header.ParentRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode header parent root `%s`", signedHeader.Header.ParentRoot)
	}

	stateRoot, err := hexutil.Decode(signedHeader.Header.StateRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode header state root `%s`", signedHeader.Header.StateRoot)
	}

	bodyRoot, err := hexutil.Decode(signedHeader.Header.BodyRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode header body root `%s`", signedHeader.Header.BodyRoot)
	}

	signature, err := hexutil.Decode(signedHeader.Signature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode signature `%s`", signedHeader.Signature)
	}

	return &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          types.Slot(slot),
			ProposerIndex: types.ValidatorIndex(proposerIndex),
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			BodyRoot:      bodyRoot,
		},
		Signature: signature,
	}, nil
}

func convertAttesterSlashingsToProto(jsonAttesterSlashings []*apimiddleware.AttesterSlashingJson) ([]*ethpb.AttesterSlashing, error) {
	attesterSlashings := make([]*ethpb.AttesterSlashing, len(jsonAttesterSlashings))

	for index, jsonAttesterSlashing := range jsonAttesterSlashings {
		if jsonAttesterSlashing == nil {
			return nil, errors.Errorf("attester slashing at index `%d` is nil", index)
		}

		attestation1, err := convertAttestationToProto(jsonAttesterSlashing.Attestation_1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get attestation 1")
		}

		attestation2, err := convertAttestationToProto(jsonAttesterSlashing.Attestation_2)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get attestation 2")
		}

		attesterSlashings[index] = &ethpb.AttesterSlashing{
			Attestation_1: attestation1,
			Attestation_2: attestation2,
		}
	}

	return attesterSlashings, nil
}

func convertAttestationToProto(jsonAttestation *apimiddleware.IndexedAttestationJson) (*ethpb.IndexedAttestation, error) {
	if jsonAttestation == nil {
		return nil, errors.New("indexed attestation is nil")
	}

	attestingIndices := make([]uint64, len(jsonAttestation.AttestingIndices))

	for index, jsonAttestingIndex := range jsonAttestation.AttestingIndices {
		attestingIndex, err := strconv.ParseUint(jsonAttestingIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse attesting index `%s`", jsonAttestingIndex)
		}

		attestingIndices[index] = attestingIndex
	}

	signature, err := hexutil.Decode(jsonAttestation.Signature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode attestation signature `%s`", jsonAttestation.Signature)
	}

	attestationData, err := convertAttestationDataToProto(jsonAttestation.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attestation data")
	}

	return &ethpb.IndexedAttestation{
		AttestingIndices: attestingIndices,
		Data:             attestationData,
		Signature:        signature,
	}, nil
}

func convertCheckpointToProto(jsonCheckpoint *apimiddleware.CheckpointJson) (*ethpb.Checkpoint, error) {
	if jsonCheckpoint == nil {
		return nil, errors.New("checkpoint is nil")
	}

	epoch, err := strconv.ParseUint(jsonCheckpoint.Epoch, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse checkpoint epoch `%s`", jsonCheckpoint.Epoch)
	}

	root, err := hexutil.Decode(jsonCheckpoint.Root)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode checkpoint root `%s`", jsonCheckpoint.Root)
	}

	return &ethpb.Checkpoint{
		Epoch: types.Epoch(epoch),
		Root:  root,
	}, nil
}

func convertAttestationsToProto(jsonAttestations []*apimiddleware.AttestationJson) ([]*ethpb.Attestation, error) {
	attestations := make([]*ethpb.Attestation, len(jsonAttestations))

	for index, jsonAttestation := range jsonAttestations {
		if jsonAttestation == nil {
			return nil, errors.Errorf("attestation at index `%d` is nil", index)
		}

		aggregationBits, err := hexutil.Decode(jsonAttestation.AggregationBits)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode aggregation bits `%s`", jsonAttestation.AggregationBits)
		}

		attestationData, err := convertAttestationDataToProto(jsonAttestation.Data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get attestation data")
		}

		signature, err := hexutil.Decode(jsonAttestation.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode attestation signature `%s`", jsonAttestation.Signature)
		}

		attestations[index] = &ethpb.Attestation{
			AggregationBits: aggregationBits,
			Data:            attestationData,
			Signature:       signature,
		}
	}

	return attestations, nil
}

func convertAttestationDataToProto(jsonAttestationData *apimiddleware.AttestationDataJson) (*ethpb.AttestationData, error) {
	if jsonAttestationData == nil {
		return nil, errors.New("attestation data is nil")
	}

	slot, err := strconv.ParseUint(jsonAttestationData.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse attestation slot `%s`", jsonAttestationData.Slot)
	}

	committeeIndex, err := strconv.ParseUint(jsonAttestationData.CommitteeIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse attestation committee index `%s`", jsonAttestationData.CommitteeIndex)
	}

	beaconBlockRoot, err := hexutil.Decode(jsonAttestationData.BeaconBlockRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode attestation beacon block root `%s`", jsonAttestationData.BeaconBlockRoot)
	}

	sourceCheckpoint, err := convertCheckpointToProto(jsonAttestationData.Source)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attestation source checkpoint")
	}

	targetCheckpoint, err := convertCheckpointToProto(jsonAttestationData.Target)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attestation target checkpoint")
	}

	return &ethpb.AttestationData{
		Slot:            types.Slot(slot),
		CommitteeIndex:  types.CommitteeIndex(committeeIndex),
		BeaconBlockRoot: beaconBlockRoot,
		Source:          sourceCheckpoint,
		Target:          targetCheckpoint,
	}, nil
}

func convertDepositsToProto(jsonDeposits []*apimiddleware.DepositJson) ([]*ethpb.Deposit, error) {
	deposits := make([]*ethpb.Deposit, len(jsonDeposits))

	for depositIndex, jsonDeposit := range jsonDeposits {
		if jsonDeposit == nil {
			return nil, errors.Errorf("deposit at index `%d` is nil", depositIndex)
		}

		proofs := make([][]byte, len(jsonDeposit.Proof))
		for proofIndex, jsonProof := range jsonDeposit.Proof {
			proof, err := hexutil.Decode(jsonProof)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to decode deposit proof `%s`", jsonProof)
			}

			proofs[proofIndex] = proof
		}

		if jsonDeposit.Data == nil {
			return nil, errors.Errorf("deposit data at index `%d` is nil", depositIndex)
		}

		pubkey, err := hexutil.Decode(jsonDeposit.Data.PublicKey)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode deposit public key `%s`", jsonDeposit.Data.PublicKey)
		}

		withdrawalCredentials, err := hexutil.Decode(jsonDeposit.Data.WithdrawalCredentials)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode deposit withdrawal credentials `%s`", jsonDeposit.Data.WithdrawalCredentials)
		}

		amount, err := strconv.ParseUint(jsonDeposit.Data.Amount, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse deposit amount `%s`", jsonDeposit.Data.Amount)
		}

		signature, err := hexutil.Decode(jsonDeposit.Data.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode signature `%s`", jsonDeposit.Data.Signature)
		}

		deposits[depositIndex] = &ethpb.Deposit{
			Proof: proofs,
			Data: &ethpb.Deposit_Data{
				PublicKey:             pubkey,
				WithdrawalCredentials: withdrawalCredentials,
				Amount:                amount,
				Signature:             signature,
			},
		}
	}

	return deposits, nil
}

func convertVoluntaryExitsToProto(jsonVoluntaryExits []*apimiddleware.SignedVoluntaryExitJson) ([]*ethpb.SignedVoluntaryExit, error) {
	attestingIndices := make([]*ethpb.SignedVoluntaryExit, len(jsonVoluntaryExits))

	for index, jsonVoluntaryExit := range jsonVoluntaryExits {
		if jsonVoluntaryExit == nil {
			return nil, errors.Errorf("signed voluntary exit at index `%d` is nil", index)
		}

		if jsonVoluntaryExit.Exit == nil {
			return nil, errors.Errorf("voluntary exit at index `%d` is nil", index)
		}

		epoch, err := strconv.ParseUint(jsonVoluntaryExit.Exit.Epoch, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse voluntary exit epoch `%s`", jsonVoluntaryExit.Exit.Epoch)
		}

		validatorIndex, err := strconv.ParseUint(jsonVoluntaryExit.Exit.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse voluntary exit validator index `%s`", jsonVoluntaryExit.Exit.ValidatorIndex)
		}

		signature, err := hexutil.Decode(jsonVoluntaryExit.Signature)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode signature `%s`", jsonVoluntaryExit.Signature)
		}

		attestingIndices[index] = &ethpb.SignedVoluntaryExit{
			Exit: &ethpb.VoluntaryExit{
				Epoch:          types.Epoch(epoch),
				ValidatorIndex: types.ValidatorIndex(validatorIndex),
			},
			Signature: signature,
		}
	}

	return attestingIndices, nil
}

func convertTransactionsToProto(jsonTransactions []string) ([][]byte, error) {
	transactions := make([][]byte, len(jsonTransactions))

	for index, jsonTransaction := range jsonTransactions {
		transaction, err := hexutil.Decode(jsonTransaction)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode transaction `%s`", jsonTransaction)
		}

		transactions[index] = transaction
	}

	return transactions, nil
}
