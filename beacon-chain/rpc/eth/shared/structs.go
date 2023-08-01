package shared

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type Attestation struct {
	AggregationBits string          `json:"aggregation_bits" validate:"required,hexadecimal"`
	Data            AttestationData `json:"data" validate:"required"`
	Signature       string          `json:"signature" validate:"required,hexadecimal"`
}

type AttestationData struct {
	Slot            string     `json:"slot" validate:"required,number,gte=0"`
	CommitteeIndex  string     `json:"index" validate:"required,number,gte=0"`
	BeaconBlockRoot string     `json:"beacon_block_root" validate:"required,hexadecimal"`
	Source          Checkpoint `json:"source" validate:"required"`
	Target          Checkpoint `json:"target" validate:"required"`
}

type Checkpoint struct {
	Epoch string `json:"epoch" validate:"required,number,gte=0"`
	Root  string `json:"root" validate:"required,hexadecimal"`
}

type SignedContributionAndProof struct {
	Message   ContributionAndProof `json:"message" validate:"required"`
	Signature string               `json:"signature" validate:"required,hexadecimal"`
}

type ContributionAndProof struct {
	AggregatorIndex string                    `json:"aggregator_index" validate:"required,number,gte=0"`
	Contribution    SyncCommitteeContribution `json:"contribution" validate:"required"`
	SelectionProof  string                    `json:"selection_proof" validate:"required,hexadecimal"`
}

type SyncCommitteeContribution struct {
	Slot              string `json:"slot" validate:"required,number,gte=0"`
	BeaconBlockRoot   string `json:"beacon_block_root" hex:"true" validate:"required,hexadecimal"`
	SubcommitteeIndex string `json:"subcommittee_index" validate:"required,number,gte=0"`
	AggregationBits   string `json:"aggregation_bits" hex:"true" validate:"required,hexadecimal"`
	Signature         string `json:"signature" hex:"true" validate:"required,hexadecimal"`
}

func (s *SignedContributionAndProof) ToConsensus() (*eth.SignedContributionAndProof, error) {
	sig, err := hexutil.Decode(s.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode s.Signature")
	}
	aggregatorIndex, err := strconv.ParseUint(s.Message.AggregatorIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode s.Message.AggregatorIndex")
	}
	selectionProof, err := hexutil.Decode(s.Message.SelectionProof)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode s.Message.SelectionProof")
	}
	slot, err := strconv.ParseUint(s.Message.Contribution.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode s.Message.Contribution.Slot")
	}
	beaconBlockRoot, err := hexutil.Decode(s.Message.Contribution.BeaconBlockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode s.Message.Contribution.BeaconBlockRoot")
	}
	subcommitteeIndex, err := strconv.ParseUint(s.Message.Contribution.SubcommitteeIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode s.Message.Contribution.SubcommitteeIndex")
	}
	aggregationBits, err := hexutil.Decode(s.Message.Contribution.AggregationBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode s.Message.Contribution.AggregationBits")
	}
	contributionSig, err := hexutil.Decode(s.Message.Contribution.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode s.Message.Contribution.Signature")
	}

	return &eth.SignedContributionAndProof{
		Message: &eth.ContributionAndProof{
			AggregatorIndex: primitives.ValidatorIndex(aggregatorIndex),
			Contribution: &eth.SyncCommitteeContribution{
				Slot:              primitives.Slot(slot),
				BlockRoot:         beaconBlockRoot,
				SubcommitteeIndex: subcommitteeIndex,
				AggregationBits:   aggregationBits,
				Signature:         contributionSig,
			},
			SelectionProof: selectionProof,
		},
		Signature: sig,
	}, nil
}
