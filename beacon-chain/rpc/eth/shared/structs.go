package shared

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type Attestation struct {
	AggregationBits string           `json:"aggregation_bits" validate:"required,hexadecimal"`
	Data            *AttestationData `json:"data" validate:"required"`
	Signature       string           `json:"signature" validate:"required,hexadecimal"`
}

type AttestationData struct {
	Slot            string      `json:"slot" validate:"required,number,gte=0"`
	CommitteeIndex  string      `json:"index" validate:"required,number,gte=0"`
	BeaconBlockRoot string      `json:"beacon_block_root" validate:"required,hexadecimal"`
	Source          *Checkpoint `json:"source" validate:"required"`
	Target          *Checkpoint `json:"target" validate:"required"`
}

type Checkpoint struct {
	Epoch string `json:"epoch" validate:"required,number,gte=0"`
	Root  string `json:"root" validate:"required,hexadecimal"`
}

type SignedContributionAndProof struct {
	Message   *ContributionAndProof `json:"message" validate:"required"`
	Signature string                `json:"signature" validate:"required,hexadecimal"`
}

type ContributionAndProof struct {
	AggregatorIndex string                     `json:"aggregator_index" validate:"required,number,gte=0"`
	Contribution    *SyncCommitteeContribution `json:"contribution" validate:"required"`
	SelectionProof  string                     `json:"selection_proof" validate:"required,hexadecimal"`
}

type SyncCommitteeContribution struct {
	Slot              string `json:"slot" validate:"required,number,gte=0"`
	BeaconBlockRoot   string `json:"beacon_block_root" hex:"true" validate:"required,hexadecimal"`
	SubcommitteeIndex string `json:"subcommittee_index" validate:"required,number,gte=0"`
	AggregationBits   string `json:"aggregation_bits" hex:"true" validate:"required,hexadecimal"`
	Signature         string `json:"signature" hex:"true" validate:"required,hexadecimal"`
}

type SignedAggregateAttestationAndProof struct {
	Message   *AggregateAttestationAndProof `json:"message" validate:"required"`
	Signature string                        `json:"signature" validate:"required,hexadecimal"`
}

type AggregateAttestationAndProof struct {
	AggregatorIndex string       `json:"aggregator_index" validate:"required,number,gte=0"`
	Aggregate       *Attestation `json:"aggregate" validate:"required"`
	SelectionProof  string       `json:"selection_proof" validate:"required,hexadecimal"`
}

type SyncCommitteeSubscription struct {
	ValidatorIndex       string   `json:"validator_index" validate:"required,number,gte=0"`
	SyncCommitteeIndices []string `json:"sync_committee_indices" validate:"required,dive,number,gte=0"`
	UntilEpoch           string   `json:"until_epoch" validate:"required,number,gte=0"`
}

type BeaconCommitteeSubscription struct {
	ValidatorIndex   string `json:"validator_index" validate:"required,number,gte=0"`
	CommitteeIndex   string `json:"committee_index" validate:"required,number,gte=0"`
	CommitteesAtSlot string `json:"committees_at_slot" validate:"required,number,gte=0"`
	Slot             string `json:"slot" validate:"required,number,gte=0"`
	IsAggregator     bool   `json:"is_aggregator"`
}

func (s *SignedContributionAndProof) ToConsensus() (*eth.SignedContributionAndProof, error) {
	msg, err := s.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}
	sig, err := hexutil.Decode(s.Signature)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}

	return &eth.SignedContributionAndProof{
		Message:   msg,
		Signature: sig,
	}, nil
}

func (c *ContributionAndProof) ToConsensus() (*eth.ContributionAndProof, error) {
	contribution, err := c.Contribution.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Contribution")
	}
	aggregatorIndex, err := strconv.ParseUint(c.AggregatorIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "AggregatorIndex")
	}
	selectionProof, err := hexutil.Decode(c.SelectionProof)
	if err != nil {
		return nil, NewDecodeError(err, "SelectionProof")
	}

	return &eth.ContributionAndProof{
		AggregatorIndex: primitives.ValidatorIndex(aggregatorIndex),
		Contribution:    contribution,
		SelectionProof:  selectionProof,
	}, nil
}

func (s *SyncCommitteeContribution) ToConsensus() (*eth.SyncCommitteeContribution, error) {
	slot, err := strconv.ParseUint(s.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	bbRoot, err := hexutil.Decode(s.BeaconBlockRoot)
	if err != nil {
		return nil, NewDecodeError(err, "BeaconBlockRoot")
	}
	subcommitteeIndex, err := strconv.ParseUint(s.SubcommitteeIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "SubcommitteeIndex")
	}
	aggBits, err := hexutil.Decode(s.AggregationBits)
	if err != nil {
		return nil, NewDecodeError(err, "AggregationBits")
	}
	sig, err := hexutil.Decode(s.Signature)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}

	return &eth.SyncCommitteeContribution{
		Slot:              primitives.Slot(slot),
		BlockRoot:         bbRoot,
		SubcommitteeIndex: subcommitteeIndex,
		AggregationBits:   aggBits,
		Signature:         sig,
	}, nil
}

func (s *SignedAggregateAttestationAndProof) ToConsensus() (*eth.SignedAggregateAttestationAndProof, error) {
	msg, err := s.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}
	sig, err := hexutil.Decode(s.Signature)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}

	return &eth.SignedAggregateAttestationAndProof{
		Message:   msg,
		Signature: sig,
	}, nil
}

func (a *AggregateAttestationAndProof) ToConsensus() (*eth.AggregateAttestationAndProof, error) {
	aggIndex, err := strconv.ParseUint(a.AggregatorIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "AggregatorIndex")
	}
	agg, err := a.Aggregate.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Aggregate")
	}
	proof, err := hexutil.Decode(a.SelectionProof)
	if err != nil {
		return nil, NewDecodeError(err, "SelectionProof")
	}
	return &eth.AggregateAttestationAndProof{
		AggregatorIndex: primitives.ValidatorIndex(aggIndex),
		Aggregate:       agg,
		SelectionProof:  proof,
	}, nil
}

func (a *Attestation) ToConsensus() (*eth.Attestation, error) {
	aggBits, err := hexutil.Decode(a.AggregationBits)
	if err != nil {
		return nil, NewDecodeError(err, "AggregationBits")
	}
	data, err := a.Data.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Data")
	}
	sig, err := hexutil.Decode(a.Signature)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}

	return &eth.Attestation{
		AggregationBits: aggBits,
		Data:            data,
		Signature:       sig,
	}, nil
}

func AttestationFromConsensus(a *eth.Attestation) *Attestation {
	return &Attestation{
		AggregationBits: hexutil.Encode(a.AggregationBits),
		Data:            AttestationDataFromConsensus(a.Data),
		Signature:       hexutil.Encode(a.Signature),
	}
}

func (a *AttestationData) ToConsensus() (*eth.AttestationData, error) {
	slot, err := strconv.ParseUint(a.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	committeeIndex, err := strconv.ParseUint(a.CommitteeIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "CommitteeIndex")
	}
	bbRoot, err := hexutil.Decode(a.BeaconBlockRoot)
	if err != nil {
		return nil, NewDecodeError(err, "BeaconBlockRoot")
	}
	source, err := a.Source.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Source")
	}
	target, err := a.Target.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Target")
	}

	return &eth.AttestationData{
		Slot:            primitives.Slot(slot),
		CommitteeIndex:  primitives.CommitteeIndex(committeeIndex),
		BeaconBlockRoot: bbRoot,
		Source:          source,
		Target:          target,
	}, nil
}

func AttestationDataFromConsensus(a *eth.AttestationData) *AttestationData {
	return &AttestationData{
		Slot:            strconv.FormatUint(uint64(a.Slot), 10),
		CommitteeIndex:  strconv.FormatUint(uint64(a.CommitteeIndex), 10),
		BeaconBlockRoot: hexutil.Encode(a.BeaconBlockRoot),
		Source:          CheckpointFromConsensus(a.Source),
		Target:          CheckpointFromConsensus(a.Target),
	}
}

func (c *Checkpoint) ToConsensus() (*eth.Checkpoint, error) {
	epoch, err := strconv.ParseUint(c.Epoch, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Epoch")
	}
	root, err := hexutil.Decode(c.Root)
	if err != nil {
		return nil, NewDecodeError(err, "Root")
	}

	return &eth.Checkpoint{
		Epoch: primitives.Epoch(epoch),
		Root:  root,
	}, nil
}

func CheckpointFromConsensus(c *eth.Checkpoint) *Checkpoint {
	return &Checkpoint{
		Epoch: strconv.FormatUint(uint64(c.Epoch), 10),
		Root:  hexutil.Encode(c.Root),
	}
}

func (s *SyncCommitteeSubscription) ToConsensus() (*validator.SyncCommitteeSubscription, error) {
	index, err := strconv.ParseUint(s.ValidatorIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ValidatorIndex")
	}
	scIndices := make([]uint64, len(s.SyncCommitteeIndices))
	for i, ix := range s.SyncCommitteeIndices {
		scIndices[i], err = strconv.ParseUint(ix, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("SyncCommitteeIndices[%d]", i))
		}
	}
	epoch, err := strconv.ParseUint(s.UntilEpoch, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "UntilEpoch")
	}

	return &validator.SyncCommitteeSubscription{
		ValidatorIndex:       primitives.ValidatorIndex(index),
		SyncCommitteeIndices: scIndices,
		UntilEpoch:           primitives.Epoch(epoch),
	}, nil
}

func (b *BeaconCommitteeSubscription) ToConsensus() (*validator.BeaconCommitteeSubscription, error) {
	valIndex, err := strconv.ParseUint(b.ValidatorIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ValidatorIndex")
	}
	committeeIndex, err := strconv.ParseUint(b.CommitteeIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "CommitteeIndex")
	}
	committeesAtSlot, err := strconv.ParseUint(b.CommitteesAtSlot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "CommitteesAtSlot")
	}
	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}

	return &validator.BeaconCommitteeSubscription{
		ValidatorIndex:   primitives.ValidatorIndex(valIndex),
		CommitteeIndex:   primitives.CommitteeIndex(committeeIndex),
		CommitteesAtSlot: committeesAtSlot,
		Slot:             primitives.Slot(slot),
		IsAggregator:     b.IsAggregator,
	}, nil
}

// SyncDetails contains information about node sync status.
type SyncDetails struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
	IsOptimistic bool   `json:"is_optimistic"`
	ElOffline    bool   `json:"el_offline"`
}

// SyncDetailsContainer is a wrapper for Data.
type SyncDetailsContainer struct {
	Data *SyncDetails `json:"data"`
}
