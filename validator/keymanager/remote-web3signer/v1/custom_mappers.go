package v1

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// All Mappings represent version 1.0 of the Web3Signer specs i.e. /api/v1/eth2

// MapForkInfo maps the eth2.ForkInfo proto to the Web3Signer spec.
func MapForkInfo(from *ethpb.Fork, genesisValidatorsRoot []byte) (*ForkInfo, error) {
	if from == nil {
		return nil, fmt.Errorf("fork info is nil")
	}
	forkData := &Fork{
		PreviousVersion: hexutil.Encode(from.PreviousVersion),
		CurrentVersion:  hexutil.Encode(from.CurrentVersion),
		Epoch:           fmt.Sprint(from.Epoch),
	}
	return &ForkInfo{
		Fork:                  forkData,
		GenesisValidatorsRoot: hexutil.Encode(genesisValidatorsRoot),
	}, nil
}

// MapAggregateAndProof maps the eth2.AggregateAndProof proto to the Web3Signer spec.
func MapAggregateAndProof(from *ethpb.AggregateAttestationAndProof) (*AggregateAndProof, error) {
	if from == nil {
		return nil, fmt.Errorf("AggregateAttestationAndProof is nil")
	}
	aggregate, err := MapAttestation(from.Aggregate)
	if err != nil {
		return nil, err
	}
	return &AggregateAndProof{
		AggregatorIndex: fmt.Sprint(from.AggregatorIndex),
		Aggregate:       aggregate,
		SelectionProof:  hexutil.Encode(from.SelectionProof),
	}, nil
}

// MapAttestation maps the eth2.Attestation proto to the Web3Signer spec.
func MapAttestation(attestation *ethpb.Attestation) (*Attestation, error) {
	if attestation == nil {
		return nil, fmt.Errorf("attestation is nil")
	}
	data, err := MapAttestationData(attestation.Data)
	if err != nil {
		return nil, err
	}
	return &Attestation{
		Data:            data,
		AggregationBits: hexutil.Encode(attestation.AggregationBits),
		Signature:       hexutil.Encode(attestation.Signature),
	}, nil
}

// MapAttestationData maps the eth2.AttestationData proto to the Web3Signer spec.
func MapAttestationData(data *ethpb.AttestationData) (*AttestationData, error) {
	if data == nil {
		return nil, fmt.Errorf("attestation data is nil")
	}
	source, err := MapCheckPoint(data.Source)
	if err != nil {
		return nil, err
	}
	target, err := MapCheckPoint(data.Target)
	if err != nil {
		return nil, err
	}
	return &AttestationData{
		Slot:            fmt.Sprint(data.Slot),
		Index:           fmt.Sprint(data.CommitteeIndex),
		BeaconBlockRoot: hexutil.Encode(data.BeaconBlockRoot),
		Source:          source,
		Target:          target,
	}, nil
}

// MapCheckPoint maps the eth2.Checkpoint proto to the Web3Signer spec.
func MapCheckPoint(checkpoint *ethpb.Checkpoint) (*Checkpoint, error) {
	if checkpoint == nil {
		return nil, fmt.Errorf("checkpoint is nil")
	}
	return &Checkpoint{
		Epoch: fmt.Sprint(checkpoint.Epoch),
		Root:  hexutil.Encode(checkpoint.Root),
	}, nil
}

// MapBeaconBlockBody maps the eth2.BeaconBlockBody proto to the Web3Signer spec.
func MapBeaconBlockBody(body *ethpb.BeaconBlockBody) (*BeaconBlockBody, error) {
	if body == nil {
		return nil, fmt.Errorf("beacon block body is nil")
	}
	block := &BeaconBlockBody{
		RandaoReveal: hexutil.Encode(body.RandaoReveal),
		Eth1Data: &Eth1Data{
			DepositRoot: hexutil.Encode(body.Eth1Data.DepositRoot),
			BlockHash:   hexutil.Encode(body.Eth1Data.BlockHash),
		},
		Graffiti:          hexutil.Encode(body.Graffiti),
		ProposerSlashings: make([]*ProposerSlashing, len(body.ProposerSlashings)),
		AttesterSlashings: make([]*AttesterSlashing, len(body.AttesterSlashings)),
		Attestations:      make([]*Attestation, len(body.Attestations)),
		Deposits:          make([]*Deposit, len(body.Deposits)),
		VoluntaryExits:    make([]*SignedVoluntaryExit, len(body.VoluntaryExits)),
	}
	for i, slashing := range body.ProposerSlashings {
		slashing, err := MapProposerSlashing(slashing)
		if err != nil {
			return nil, err
		}
		block.ProposerSlashings[i] = slashing
	}
	for i, slashing := range body.AttesterSlashings {
		slashing, err := MapAttesterSlashing(slashing)
		if err != nil {
			return nil, err
		}
		block.AttesterSlashings[i] = slashing
	}
	for i, attestation := range body.Attestations {
		attestation, err := MapAttestation(attestation)
		if err != nil {
			return nil, err
		}
		block.Attestations[i] = attestation
	}
	for i, Deposit := range body.Deposits {
		deposit, err := MapDeposit(Deposit)
		if err != nil {
			return nil, err
		}
		block.Deposits[i] = deposit
	}
	for i, signedVoluntaryExit := range body.VoluntaryExits {
		signedVoluntaryExit, err := MapSignedVoluntaryExit(signedVoluntaryExit)
		if err != nil {
			return nil, err
		}
		block.VoluntaryExits[i] = signedVoluntaryExit
	}
	return block, nil
}

// MapProposerSlashing maps the eth2.ProposerSlashing proto to the Web3Signer spec.
func MapProposerSlashing(slashing *ethpb.ProposerSlashing) (*ProposerSlashing, error) {
	if slashing == nil {
		return nil, fmt.Errorf("proposer slashing is nil")
	}
	signedHeader1, err := MapSignedBeaconBlockHeader(slashing.Header_1)
	if err != nil {
		return nil, err
	}
	signedHeader2, err := MapSignedBeaconBlockHeader(slashing.Header_2)
	if err != nil {
		return nil, err
	}
	return &ProposerSlashing{
		SignedHeader_1: signedHeader1,
		SignedHeader_2: signedHeader2,
	}, nil
}

func MapSignedBeaconBlockHeader(signedHeader *ethpb.SignedBeaconBlockHeader) (*SignedBeaconBlockHeader, error) {
	if signedHeader == nil {
		return nil, fmt.Errorf("signed beacon block header is nil")
	}
	return &SignedBeaconBlockHeader{
		Message: &BeaconBlockHeader{
			Slot:          fmt.Sprint(signedHeader.Header.Slot),
			ProposerIndex: fmt.Sprint(signedHeader.Header.ProposerIndex),
			ParentRoot: hexutil.Encode(
				signedHeader.Header.ParentRoot,
			),
			StateRoot: hexutil.Encode(
				signedHeader.Header.StateRoot,
			),
			BodyRoot: hexutil.Encode(
				signedHeader.Header.BodyRoot,
			),
		},
		Signature: hexutil.Encode(
			signedHeader.Signature,
		),
	}, nil
}

func MapAttesterSlashing(slashing *ethpb.AttesterSlashing) (*AttesterSlashing, error) {
	if slashing == nil {
		return nil, fmt.Errorf("attester slashing is nil")
	}
	attestation1, err := MapIndexedAttestation(slashing.Attestation_1)
	if err != nil {
		return nil, err
	}
	attestation2, err := MapIndexedAttestation(slashing.Attestation_2)
	if err != nil {
		return nil, err
	}
	return &AttesterSlashing{
		Attestation_1: attestation1,
		Attestation_2: attestation2,
	}, nil
}

func MapIndexedAttestation(attestation *ethpb.IndexedAttestation) (*IndexedAttestation, error) {
	if attestation == nil {
		return nil, fmt.Errorf("indexed attestation is nil")
	}
	attestingIndices := make([]string, len(attestation.AttestingIndices))
	for i, indices := range attestation.AttestingIndices {
		attestingIndices[i] = fmt.Sprint(indices)
	}
	attestationData, err := MapAttestationData(attestation.Data)
	if err != nil {
		return nil, err
	}
	return &IndexedAttestation{
		AttestingIndices: attestingIndices,
		Data:             attestationData,
	}, nil
}

func MapDeposit(deposit *ethpb.Deposit) (*Deposit, error) {
	if deposit == nil {
		return nil, fmt.Errorf("deposit is nil")
	}
	proof := make([]string, len(deposit.Proof))
	for i, p := range proof {
		proof[i] = fmt.Sprint(p)
	}
	return &Deposit{
		Proof: proof,
		Data: &DepositData{
			PublicKey: hexutil.Encode(
				deposit.Data.PublicKey,
			),
			WithdrawalCredentials: hexutil.Encode(
				deposit.Data.WithdrawalCredentials,
			),
			Amount: fmt.Sprint(deposit.Data.Amount),
			Signature: hexutil.Encode(
				deposit.Data.Signature,
			),
		},
	}, nil
}

func MapSignedVoluntaryExit(signedVoluntaryExit *ethpb.SignedVoluntaryExit) (*SignedVoluntaryExit, error) {
	if signedVoluntaryExit == nil {
		return nil, fmt.Errorf("signed voluntary exit is nil")
	}
	return &SignedVoluntaryExit{
		Message: &VoluntaryExit{
			Epoch:          fmt.Sprint(signedVoluntaryExit.Exit.Epoch),
			ValidatorIndex: fmt.Sprint(signedVoluntaryExit.Exit.ValidatorIndex),
		},
		Signature: hexutil.Encode(
			signedVoluntaryExit.Signature,
		),
	}, nil
}
