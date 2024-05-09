package encoder_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"testing"

	gogo "github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	"google.golang.org/protobuf/proto"
)

type ProtoCreator interface {
	Create() proto.Message
}

type AttestationCreator struct{}
type AttestationElectraCreator struct{}
type AggregateAttestationAndProofCreator struct{}
type AggregateAttestationAndProofElectraCreator struct{}
type SignedAggregateAttestationAndProofCreator struct{}
type SignedAggregateAttestationAndProofElectraCreator struct{}
type AttestationDataCreator struct{}
type CheckpointCreator struct{}
type BeaconBlockCreator struct{}
type SignedBeaconBlockCreator struct{}
type BeaconBlockAltairCreator struct{}
type SignedBeaconBlockAltairCreator struct{}
type BeaconBlockBodyCreator struct{}
type BeaconBlockBodyAltairCreator struct{}
type ProposerSlashingCreator struct{}
type AttesterSlashingCreator struct{}
type AttesterSlashingElectraCreator struct{}
type DepositCreator struct{}
type VoluntaryExitCreator struct{}
type SignedVoluntaryExitCreator struct{}
type Eth1DataCreator struct{}
type BeaconBlockHeaderCreator struct{}
type SignedBeaconBlockHeaderCreator struct{}
type IndexedAttestationCreator struct{}
type IndexedAttestationElectraCreator struct{}
type SyncAggregateCreator struct{}
type SignedBeaconBlockBellatrixCreator struct{}
type BeaconBlockBellatrixCreator struct{}
type BeaconBlockBodyBellatrixCreator struct{}
type SignedBlindedBeaconBlockBellatrixCreator struct{}
type BlindedBeaconBlockBellatrixCreator struct{}
type BlindedBeaconBlockBodyBellatrixCreator struct{}
type SignedBeaconBlockContentsDenebCreator struct{}
type BeaconBlockContentsDenebCreator struct{}
type SignedBeaconBlockDenebCreator struct{}
type BeaconBlockDenebCreator struct{}
type BeaconBlockBodyDenebCreator struct{}
type SignedBeaconBlockCapellaCreator struct{}
type BeaconBlockCapellaCreator struct{}
type BeaconBlockBodyCapellaCreator struct{}
type SignedBlindedBeaconBlockCapellaCreator struct{}
type BlindedBeaconBlockCapellaCreator struct{}
type BlindedBeaconBlockBodyCapellaCreator struct{}
type SignedBlindedBeaconBlockDenebCreator struct{}
type BlindedBeaconBlockDenebCreator struct{}
type BlindedBeaconBlockBodyDenebCreator struct{}
type SignedBeaconBlockElectraCreator struct{}
type BeaconBlockElectraCreator struct{}
type BeaconBlockBodyElectraCreator struct{}
type SignedBlindedBeaconBlockElectraCreator struct{}
type BlindedBeaconBlockElectraCreator struct{}
type BlindedBeaconBlockBodyElectraCreator struct{}
type ValidatorRegistrationV1Creator struct{}
type SignedValidatorRegistrationV1Creator struct{}
type BuilderBidCreator struct{}
type BuilderBidCapellaCreator struct{}
type BuilderBidDenebCreator struct{}
type BlobSidecarCreator struct{}
type BlobSidecarsCreator struct{}
type Deposit_DataCreator struct{}
type BeaconStateCreator struct{}
type BeaconStateAltairCreator struct{}
type ForkCreator struct{}
type PendingAttestationCreator struct{}
type HistoricalBatchCreator struct{}
type SigningDataCreator struct{}
type ForkDataCreator struct{}
type DepositMessageCreator struct{}
type SyncCommitteeCreator struct{}
type SyncAggregatorSelectionDataCreator struct{}
type BeaconStateBellatrixCreator struct{}
type BeaconStateCapellaCreator struct{}
type BeaconStateDenebCreator struct{}
type BeaconStateElectraCreator struct{}
type PowBlockCreator struct{}
type HistoricalSummaryCreator struct{}
type BlobIdentifierCreator struct{}
type PendingBalanceDepositCreator struct{}
type PendingPartialWithdrawalCreator struct{}
type ConsolidationCreator struct{}
type SignedConsolidationCreator struct{}
type PendingConsolidationCreator struct{}
type StatusCreator struct{}
type BeaconBlocksByRangeRequestCreator struct{}
type ENRForkIDCreator struct{}
type MetaDataV0Creator struct{}
type MetaDataV1Creator struct{}
type BlobSidecarsByRangeRequestCreator struct{}
type DepositSnapshotCreator struct{}
type SyncCommitteeMessageCreator struct{}
type SyncCommitteeContributionCreator struct{}
type ContributionAndProofCreator struct{}
type SignedContributionAndProofCreator struct{}
type ValidatorCreator struct{}
type BLSToExecutionChangeCreator struct{}
type SignedBLSToExecutionChangeCreator struct{}

func (a AttestationCreator) Create() proto.Message        { return &ethpb.Attestation{} }
func (a AttestationElectraCreator) Create() proto.Message { return &ethpb.AttestationElectra{} }
func (a AggregateAttestationAndProofCreator) Create() proto.Message {
	return &ethpb.AggregateAttestationAndProof{}
}
func (a AggregateAttestationAndProofElectraCreator) Create() proto.Message {
	return &ethpb.AggregateAttestationAndProofElectra{}
}
func (a SignedAggregateAttestationAndProofCreator) Create() proto.Message {
	return &ethpb.SignedAggregateAttestationAndProof{}
}
func (a SignedAggregateAttestationAndProofElectraCreator) Create() proto.Message {
	return &ethpb.SignedAggregateAttestationAndProofElectra{}
}
func (a AttestationDataCreator) Create() proto.Message   { return &ethpb.AttestationData{} }
func (a CheckpointCreator) Create() proto.Message        { return &ethpb.Checkpoint{} }
func (b BeaconBlockCreator) Create() proto.Message       { return &ethpb.BeaconBlock{} }
func (b SignedBeaconBlockCreator) Create() proto.Message { return &ethpb.SignedBeaconBlock{} }
func (b BeaconBlockAltairCreator) Create() proto.Message { return &ethpb.BeaconBlockAltair{} }
func (b SignedBeaconBlockAltairCreator) Create() proto.Message {
	return &ethpb.SignedBeaconBlockAltair{}
}
func (b BeaconBlockBodyCreator) Create() proto.Message       { return &ethpb.BeaconBlockBody{} }
func (b BeaconBlockBodyAltairCreator) Create() proto.Message { return &ethpb.BeaconBlockBodyAltair{} }
func (b ProposerSlashingCreator) Create() proto.Message      { return &ethpb.ProposerSlashing{} }
func (b AttesterSlashingCreator) Create() proto.Message      { return &ethpb.AttesterSlashing{} }
func (b AttesterSlashingElectraCreator) Create() proto.Message {
	return &ethpb.AttesterSlashingElectra{}
}
func (b DepositCreator) Create() proto.Message             { return &ethpb.Deposit{} }
func (b VoluntaryExitCreator) Create() proto.Message       { return &ethpb.VoluntaryExit{} }
func (b SignedVoluntaryExitCreator) Create() proto.Message { return &ethpb.SignedVoluntaryExit{} }
func (b Eth1DataCreator) Create() proto.Message            { return &ethpb.Eth1Data{} }
func (b BeaconBlockHeaderCreator) Create() proto.Message   { return &ethpb.BeaconBlockHeader{} }
func (b SignedBeaconBlockHeaderCreator) Create() proto.Message {
	return &ethpb.SignedBeaconBlockHeader{}
}
func (b IndexedAttestationCreator) Create() proto.Message { return &ethpb.IndexedAttestation{} }
func (b IndexedAttestationElectraCreator) Create() proto.Message {
	return &ethpb.IndexedAttestationElectra{}
}
func (b SyncAggregateCreator) Create() proto.Message { return &ethpb.SyncAggregate{} }
func (b SignedBeaconBlockBellatrixCreator) Create() proto.Message {
	return &ethpb.SignedBeaconBlockBellatrix{}
}
func (b BeaconBlockBellatrixCreator) Create() proto.Message { return &ethpb.BeaconBlockBellatrix{} }
func (b BeaconBlockBodyBellatrixCreator) Create() proto.Message {
	return &ethpb.BeaconBlockBodyBellatrix{}
}
func (b SignedBlindedBeaconBlockBellatrixCreator) Create() proto.Message {
	return &ethpb.SignedBlindedBeaconBlockBellatrix{}
}
func (b BlindedBeaconBlockBellatrixCreator) Create() proto.Message {
	return &ethpb.BlindedBeaconBlockBellatrix{}
}
func (b BlindedBeaconBlockBodyBellatrixCreator) Create() proto.Message {
	return &ethpb.BlindedBeaconBlockBodyBellatrix{}
}
func (b SignedBeaconBlockContentsDenebCreator) Create() proto.Message {
	return &ethpb.SignedBeaconBlockContentsDeneb{}
}
func (b BeaconBlockContentsDenebCreator) Create() proto.Message {
	return &ethpb.BeaconBlockContentsDeneb{}
}
func (b SignedBeaconBlockDenebCreator) Create() proto.Message { return &ethpb.SignedBeaconBlockDeneb{} }
func (b BeaconBlockDenebCreator) Create() proto.Message       { return &ethpb.BeaconBlockDeneb{} }
func (b BeaconBlockBodyDenebCreator) Create() proto.Message   { return &ethpb.BeaconBlockBodyDeneb{} }
func (b SignedBeaconBlockCapellaCreator) Create() proto.Message {
	return &ethpb.SignedBeaconBlockCapella{}
}
func (b BeaconBlockCapellaCreator) Create() proto.Message     { return &ethpb.BeaconBlockCapella{} }
func (b BeaconBlockBodyCapellaCreator) Create() proto.Message { return &ethpb.BeaconBlockBodyCapella{} }
func (b SignedBlindedBeaconBlockCapellaCreator) Create() proto.Message {
	return &ethpb.SignedBlindedBeaconBlockCapella{}
}
func (b BlindedBeaconBlockCapellaCreator) Create() proto.Message {
	return &ethpb.BlindedBeaconBlockCapella{}
}
func (b BlindedBeaconBlockBodyCapellaCreator) Create() proto.Message {
	return &ethpb.BlindedBeaconBlockBodyCapella{}
}
func (b SignedBlindedBeaconBlockDenebCreator) Create() proto.Message {
	return &ethpb.SignedBlindedBeaconBlockDeneb{}
}
func (b BlindedBeaconBlockDenebCreator) Create() proto.Message {
	return &ethpb.BlindedBeaconBlockDeneb{}
}
func (b BlindedBeaconBlockBodyDenebCreator) Create() proto.Message {
	return &ethpb.BlindedBeaconBlockBodyDeneb{}
}
func (b SignedBeaconBlockElectraCreator) Create() proto.Message {
	return &ethpb.SignedBeaconBlockElectra{}
}
func (b BeaconBlockElectraCreator) Create() proto.Message     { return &ethpb.BeaconBlockElectra{} }
func (b BeaconBlockBodyElectraCreator) Create() proto.Message { return &ethpb.BeaconBlockBodyElectra{} }
func (b SignedBlindedBeaconBlockElectraCreator) Create() proto.Message {
	return &ethpb.SignedBlindedBeaconBlockElectra{}
}
func (b BlindedBeaconBlockElectraCreator) Create() proto.Message {
	return &ethpb.BlindedBeaconBlockElectra{}
}
func (b BlindedBeaconBlockBodyElectraCreator) Create() proto.Message {
	return &ethpb.BlindedBeaconBlockBodyElectra{}
}
func (b ValidatorRegistrationV1Creator) Create() proto.Message {
	return &ethpb.ValidatorRegistrationV1{}
}
func (b SignedValidatorRegistrationV1Creator) Create() proto.Message {
	return &ethpb.SignedValidatorRegistrationV1{}
}
func (b BuilderBidCreator) Create() proto.Message         { return &ethpb.BuilderBid{} }
func (b BuilderBidCapellaCreator) Create() proto.Message  { return &ethpb.BuilderBidCapella{} }
func (b BuilderBidDenebCreator) Create() proto.Message    { return &ethpb.BuilderBidDeneb{} }
func (b BlobSidecarCreator) Create() proto.Message        { return &ethpb.BlobSidecar{} }
func (b BlobSidecarsCreator) Create() proto.Message       { return &ethpb.BlobSidecars{} }
func (b Deposit_DataCreator) Create() proto.Message       { return &ethpb.Deposit_Data{} }
func (b BeaconStateCreator) Create() proto.Message        { return &ethpb.BeaconState{} }
func (b BeaconStateAltairCreator) Create() proto.Message  { return &ethpb.BeaconStateAltair{} }
func (b ForkCreator) Create() proto.Message               { return &ethpb.Fork{} }
func (b PendingAttestationCreator) Create() proto.Message { return &ethpb.PendingAttestation{} }
func (b HistoricalBatchCreator) Create() proto.Message    { return &ethpb.HistoricalBatch{} }
func (b SigningDataCreator) Create() proto.Message        { return &ethpb.SigningData{} }
func (b ForkDataCreator) Create() proto.Message           { return &ethpb.ForkData{} }
func (b DepositMessageCreator) Create() proto.Message     { return &ethpb.DepositMessage{} }
func (b SyncCommitteeCreator) Create() proto.Message      { return &ethpb.SyncCommittee{} }
func (b SyncAggregatorSelectionDataCreator) Create() proto.Message {
	return &ethpb.SyncAggregatorSelectionData{}
}
func (b BeaconStateBellatrixCreator) Create() proto.Message  { return &ethpb.BeaconStateBellatrix{} }
func (b BeaconStateCapellaCreator) Create() proto.Message    { return &ethpb.BeaconStateCapella{} }
func (b BeaconStateDenebCreator) Create() proto.Message      { return &ethpb.BeaconStateDeneb{} }
func (b BeaconStateElectraCreator) Create() proto.Message    { return &ethpb.BeaconStateElectra{} }
func (b PowBlockCreator) Create() proto.Message              { return &ethpb.PowBlock{} }
func (b HistoricalSummaryCreator) Create() proto.Message     { return &ethpb.HistoricalSummary{} }
func (b BlobIdentifierCreator) Create() proto.Message        { return &ethpb.BlobIdentifier{} }
func (b PendingBalanceDepositCreator) Create() proto.Message { return &ethpb.PendingBalanceDeposit{} }
func (b PendingPartialWithdrawalCreator) Create() proto.Message {
	return &ethpb.PendingPartialWithdrawal{}
}
func (b ConsolidationCreator) Create() proto.Message        { return &ethpb.Consolidation{} }
func (b SignedConsolidationCreator) Create() proto.Message  { return &ethpb.SignedConsolidation{} }
func (b PendingConsolidationCreator) Create() proto.Message { return &ethpb.PendingConsolidation{} }
func (b StatusCreator) Create() proto.Message               { return &ethpb.Status{} }
func (b BeaconBlocksByRangeRequestCreator) Create() proto.Message {
	return &ethpb.BeaconBlocksByRangeRequest{}
}
func (b ENRForkIDCreator) Create() proto.Message  { return &ethpb.ENRForkID{} }
func (b MetaDataV0Creator) Create() proto.Message { return &ethpb.MetaDataV0{} }
func (b MetaDataV1Creator) Create() proto.Message { return &ethpb.MetaDataV1{} }
func (b BlobSidecarsByRangeRequestCreator) Create() proto.Message {
	return &ethpb.BlobSidecarsByRangeRequest{}
}
func (b DepositSnapshotCreator) Create() proto.Message      { return &ethpb.DepositSnapshot{} }
func (b SyncCommitteeMessageCreator) Create() proto.Message { return &ethpb.SyncCommitteeMessage{} }
func (b SyncCommitteeContributionCreator) Create() proto.Message {
	return &ethpb.SyncCommitteeContribution{}
}
func (b ContributionAndProofCreator) Create() proto.Message { return &ethpb.ContributionAndProof{} }
func (b SignedContributionAndProofCreator) Create() proto.Message {
	return &ethpb.SignedContributionAndProof{}
}
func (b ValidatorCreator) Create() proto.Message            { return &ethpb.Validator{} }
func (b BLSToExecutionChangeCreator) Create() proto.Message { return &ethpb.BLSToExecutionChange{} }
func (b SignedBLSToExecutionChangeCreator) Create() proto.Message {
	return &ethpb.SignedBLSToExecutionChange{}
}

var creators = []ProtoCreator{
	AttestationCreator{},
	AttestationElectraCreator{},
	AggregateAttestationAndProofCreator{},
	AggregateAttestationAndProofElectraCreator{},
	SignedAggregateAttestationAndProofCreator{},
	SignedAggregateAttestationAndProofElectraCreator{},
	AttestationDataCreator{},
	CheckpointCreator{},
	BeaconBlockCreator{},
	SignedBeaconBlockCreator{},
	BeaconBlockAltairCreator{},
	SignedBeaconBlockAltairCreator{},
	BeaconBlockBodyCreator{},
	BeaconBlockBodyAltairCreator{},
	ProposerSlashingCreator{},
	AttesterSlashingCreator{},
	AttesterSlashingElectraCreator{},
	DepositCreator{},
	VoluntaryExitCreator{},
	SignedVoluntaryExitCreator{},
	Eth1DataCreator{},
	BeaconBlockHeaderCreator{},
	SignedBeaconBlockHeaderCreator{},
	IndexedAttestationCreator{},
	IndexedAttestationElectraCreator{},
	SyncAggregateCreator{},
	SignedBeaconBlockBellatrixCreator{},
	BeaconBlockBellatrixCreator{},
	BeaconBlockBodyBellatrixCreator{},
	SignedBlindedBeaconBlockBellatrixCreator{},
	BlindedBeaconBlockBellatrixCreator{},
	BlindedBeaconBlockBodyBellatrixCreator{},
	SignedBeaconBlockContentsDenebCreator{},
	BeaconBlockContentsDenebCreator{},
	SignedBeaconBlockDenebCreator{},
	BeaconBlockDenebCreator{},
	BeaconBlockBodyDenebCreator{},
	SignedBeaconBlockCapellaCreator{},
	BeaconBlockCapellaCreator{},
	BeaconBlockBodyCapellaCreator{},
	SignedBlindedBeaconBlockCapellaCreator{},
	BlindedBeaconBlockCapellaCreator{},
	BlindedBeaconBlockBodyCapellaCreator{},
	SignedBlindedBeaconBlockDenebCreator{},
	BlindedBeaconBlockDenebCreator{},
	BlindedBeaconBlockBodyDenebCreator{},
	SignedBeaconBlockElectraCreator{},
	BeaconBlockElectraCreator{},
	BeaconBlockBodyElectraCreator{},
	SignedBlindedBeaconBlockElectraCreator{},
	BlindedBeaconBlockElectraCreator{},
	BlindedBeaconBlockBodyElectraCreator{},
	ValidatorRegistrationV1Creator{},
	SignedValidatorRegistrationV1Creator{},
	BuilderBidCreator{},
	BuilderBidCapellaCreator{},
	BuilderBidDenebCreator{},
	BlobSidecarCreator{},
	BlobSidecarsCreator{},
	Deposit_DataCreator{},
	BeaconStateCreator{},
	BeaconStateAltairCreator{},
	ForkCreator{},
	PendingAttestationCreator{},
	HistoricalBatchCreator{},
	SigningDataCreator{},
	ForkDataCreator{},
	DepositMessageCreator{},
	SyncCommitteeCreator{},
	SyncAggregatorSelectionDataCreator{},
	BeaconStateBellatrixCreator{},
	BeaconStateCapellaCreator{},
	BeaconStateDenebCreator{},
	BeaconStateElectraCreator{},
	PowBlockCreator{},
	HistoricalSummaryCreator{},
	BlobIdentifierCreator{},
	PendingBalanceDepositCreator{},
	PendingPartialWithdrawalCreator{},
	ConsolidationCreator{},
	SignedConsolidationCreator{},
	PendingConsolidationCreator{},
	StatusCreator{},
	BeaconBlocksByRangeRequestCreator{},
	ENRForkIDCreator{},
	MetaDataV0Creator{},
	MetaDataV1Creator{},
	BlobSidecarsByRangeRequestCreator{},
	DepositSnapshotCreator{},
	SyncCommitteeMessageCreator{},
	SyncCommitteeContributionCreator{},
	ContributionAndProofCreator{},
	SignedContributionAndProofCreator{},
	ValidatorCreator{},
	BLSToExecutionChangeCreator{},
	SignedBLSToExecutionChangeCreator{},
}

func FuzzRoundTripWithGossip(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte, index int) {
		if index < 0 {
			t.Skip()
		}
		// Select a random creator from the list.
		creator := creators[index%len(creators)]
		msg := creator.Create()

		e := &encoder.SszNetworkEncoder{}
		buf := new(bytes.Buffer)

		if err := proto.Unmarshal(data, msg); err != nil {
			t.Logf("Failed to unmarshal: %v", err)
			return
		}

		switch msg := msg.(type) {
		case *ethpb.Attestation:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.AttestationElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.AggregateAttestationAndProof:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.AggregateAttestationAndProofElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedAggregateAttestationAndProof:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedAggregateAttestationAndProofElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.AttestationData:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.Checkpoint:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlock:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBeaconBlock:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockAltair:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBeaconBlockAltair:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockBody:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockBodyAltair:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.ProposerSlashing:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.AttesterSlashing:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.AttesterSlashingElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.Deposit:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.VoluntaryExit:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedVoluntaryExit:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.Eth1Data:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockHeader:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBeaconBlockHeader:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.IndexedAttestation:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.IndexedAttestationElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SyncAggregate:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBeaconBlockBellatrix:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockBellatrix:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockBodyBellatrix:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBlindedBeaconBlockBellatrix:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlindedBeaconBlockBellatrix:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlindedBeaconBlockBodyBellatrix:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBeaconBlockContentsDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockContentsDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBeaconBlockDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockBodyDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBeaconBlockCapella:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockCapella:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockBodyCapella:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBlindedBeaconBlockCapella:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlindedBeaconBlockCapella:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlindedBeaconBlockBodyCapella:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBlindedBeaconBlockDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlindedBeaconBlockDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlindedBeaconBlockBodyDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBeaconBlockElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlockBodyElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBlindedBeaconBlockElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlindedBeaconBlockElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlindedBeaconBlockBodyElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.ValidatorRegistrationV1:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedValidatorRegistrationV1:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BuilderBid:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BuilderBidCapella:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BuilderBidDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlobSidecar:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlobSidecars:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.Deposit_Data:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconState:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconStateAltair:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.Fork:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.PendingAttestation:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.HistoricalBatch:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SigningData:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.ForkData:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.DepositMessage:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SyncCommittee:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SyncAggregatorSelectionData:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconStateBellatrix:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconStateCapella:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconStateDeneb:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconStateElectra:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.PowBlock:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.HistoricalSummary:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlobIdentifier:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.PendingBalanceDeposit:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.PendingPartialWithdrawal:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.Consolidation:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedConsolidation:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.PendingConsolidation:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.Status:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BeaconBlocksByRangeRequest:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.ENRForkID:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.MetaDataV0:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.MetaDataV1:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BlobSidecarsByRangeRequest:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.DepositSnapshot:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SyncCommitteeMessage:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SyncCommitteeContribution:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.ContributionAndProof:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedContributionAndProof:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.Validator:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.BLSToExecutionChange:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		case *ethpb.SignedBLSToExecutionChange:
			_, err := e.EncodeGossip(buf, msg)
			if err != nil {
				t.Logf("Failed to encode: %v", err)
				return
			}
		default:
			t.Fatalf("Unknown type: %T", msg)
		}

		decoded := creator.Create()

		// Use type assertion to handle decoded based on its type
		switch decoded := decoded.(type) {
		case *ethpb.Attestation:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.AttestationElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.AggregateAttestationAndProof:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.AggregateAttestationAndProofElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedAggregateAttestationAndProof:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedAggregateAttestationAndProofElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.AttestationData:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.Checkpoint:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlock:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBeaconBlock:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockAltair:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBeaconBlockAltair:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockBody:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockBodyAltair:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.ProposerSlashing:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.AttesterSlashing:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.AttesterSlashingElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.Deposit:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.VoluntaryExit:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedVoluntaryExit:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.Eth1Data:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockHeader:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBeaconBlockHeader:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.IndexedAttestation:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.IndexedAttestationElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SyncAggregate:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBeaconBlockBellatrix:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockBellatrix:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockBodyBellatrix:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBlindedBeaconBlockBellatrix:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlindedBeaconBlockBellatrix:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlindedBeaconBlockBodyBellatrix:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBeaconBlockContentsDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockContentsDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBeaconBlockDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockBodyDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBeaconBlockCapella:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockCapella:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockBodyCapella:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBlindedBeaconBlockCapella:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlindedBeaconBlockCapella:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlindedBeaconBlockBodyCapella:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBlindedBeaconBlockDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlindedBeaconBlockDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlindedBeaconBlockBodyDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBeaconBlockElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlockBodyElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBlindedBeaconBlockElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlindedBeaconBlockElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlindedBeaconBlockBodyElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.ValidatorRegistrationV1:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedValidatorRegistrationV1:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BuilderBid:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BuilderBidCapella:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BuilderBidDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlobSidecar:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlobSidecars:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.Deposit_Data:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconState:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconStateAltair:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.Fork:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.PendingAttestation:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.HistoricalBatch:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SigningData:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.ForkData:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.DepositMessage:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SyncCommittee:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SyncAggregatorSelectionData:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconStateBellatrix:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconStateCapella:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconStateDeneb:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconStateElectra:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.PowBlock:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.HistoricalSummary:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlobIdentifier:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.PendingBalanceDeposit:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.PendingPartialWithdrawal:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.Consolidation:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedConsolidation:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.PendingConsolidation:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.Status:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BeaconBlocksByRangeRequest:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.ENRForkID:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.MetaDataV0:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.MetaDataV1:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BlobSidecarsByRangeRequest:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.DepositSnapshot:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SyncCommitteeMessage:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SyncCommitteeContribution:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.ContributionAndProof:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedContributionAndProof:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.Validator:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.BLSToExecutionChange:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		case *ethpb.SignedBLSToExecutionChange:
			if err := e.DecodeGossip(buf.Bytes(), decoded); err != nil {
				t.Fatalf("Failed to decode: %v", err)
			}
		default:
			t.Fatalf("Unknown type: %T", decoded)
		}

		if !proto.Equal(decoded, msg) {
			t.Logf("Decoded message: %#+v is not the same as original: %#+v", decoded, msg)
		}
	})
}

func TestSszNetworkEncoder_RoundTrip_Consolidation(t *testing.T) {
	e := &encoder.SszNetworkEncoder{}
	buf := new(bytes.Buffer)

	data := []byte("\xc800")
	msg := &ethpb.Consolidation{}

	if err := proto.Unmarshal(data, msg); err != nil {
		t.Logf("Failed to unmarshal: %v", err)
		return
	}

	fmt.Printf("msg=%#+v\n", msg)
	_, err := e.EncodeGossip(buf, msg)
	require.NoError(t, err)
	decoded := &ethpb.Consolidation{}
	require.NoError(t, e.DecodeGossip(buf.Bytes(), decoded))

	if !proto.Equal(decoded, msg) {
		fmt.Printf("decoded=%#+v\n", decoded)
		t.Logf("decoded=%+v\n", decoded)
		fmt.Printf("msg=%+v\n", msg)
		t.Logf("msg=%+v\n", msg)
		//t.Error("Decoded message is not the same as original")
	}
}

func TestSszNetworkEncoder_RoundTrip(t *testing.T) {
	e := &encoder.SszNetworkEncoder{}
	testRoundTripWithLength(t, e)
	testRoundTripWithGossip(t, e)
}

func TestSszNetworkEncoder_FailsSnappyLength(t *testing.T) {
	e := &encoder.SszNetworkEncoder{}
	att := &ethpb.Fork{}
	data := make([]byte, 32)
	binary.PutUvarint(data, encoder.MaxGossipSize+32)
	err := e.DecodeGossip(data, att)
	require.ErrorContains(t, "snappy message exceeds max size", err)
}

func testRoundTripWithLength(t *testing.T, e *encoder.SszNetworkEncoder) {
	buf := new(bytes.Buffer)
	msg := &ethpb.Fork{
		PreviousVersion: []byte("fooo"),
		CurrentVersion:  []byte("barr"),
		Epoch:           9001,
	}
	_, err := e.EncodeWithMaxLength(buf, msg)
	require.NoError(t, err)
	decoded := &ethpb.Fork{}
	require.NoError(t, e.DecodeWithMaxLength(buf, decoded))
	if !proto.Equal(decoded, msg) {
		t.Logf("decoded=%+v\n", decoded)
		t.Error("Decoded message is not the same as original")
	}
}

func testRoundTripWithGossip(t *testing.T, e *encoder.SszNetworkEncoder) {
	buf := new(bytes.Buffer)
	msg := &ethpb.Fork{
		PreviousVersion: []byte("fooo"),
		CurrentVersion:  []byte("barr"),
		Epoch:           9001,
	}
	_, err := e.EncodeGossip(buf, msg)
	require.NoError(t, err)
	decoded := &ethpb.Fork{}
	require.NoError(t, e.DecodeGossip(buf.Bytes(), decoded))
	if !proto.Equal(decoded, msg) {
		t.Logf("decoded=%+v\n", decoded)
		t.Error("Decoded message is not the same as original")
	}
}

func TestSszNetworkEncoder_EncodeWithMaxLength(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := &ethpb.Fork{
		PreviousVersion: []byte("fooo"),
		CurrentVersion:  []byte("barr"),
		Epoch:           9001,
	}
	e := &encoder.SszNetworkEncoder{}
	params.SetupTestConfigCleanup(t)
	c := params.BeaconNetworkConfig()
	encoder.MaxChunkSize = uint64(5)
	params.OverrideBeaconNetworkConfig(c)
	_, err := e.EncodeWithMaxLength(buf, msg)
	wanted := fmt.Sprintf("which is larger than the provided max limit of %d", encoder.MaxChunkSize)
	assert.ErrorContains(t, wanted, err)
}

func TestSszNetworkEncoder_DecodeWithMaxLength(t *testing.T) {
	buf := new(bytes.Buffer)
	msg := &ethpb.Fork{
		PreviousVersion: []byte("fooo"),
		CurrentVersion:  []byte("barr"),
		Epoch:           4242,
	}
	e := &encoder.SszNetworkEncoder{}
	params.SetupTestConfigCleanup(t)
	c := params.BeaconNetworkConfig()
	maxChunkSize := uint64(5)
	encoder.MaxChunkSize = maxChunkSize
	params.OverrideBeaconNetworkConfig(c)
	_, err := e.EncodeGossip(buf, msg)
	require.NoError(t, err)
	decoded := &ethpb.Fork{}
	err = e.DecodeWithMaxLength(buf, decoded)
	wanted := fmt.Sprintf("goes over the provided max limit of %d", maxChunkSize)
	assert.ErrorContains(t, wanted, err)
}

func TestSszNetworkEncoder_DecodeWithMultipleFrames(t *testing.T) {
	buf := new(bytes.Buffer)
	st, _ := util.DeterministicGenesisState(t, 100)
	e := &encoder.SszNetworkEncoder{}
	params.SetupTestConfigCleanup(t)
	c := params.BeaconNetworkConfig()
	// 4 * 1 Mib
	maxChunkSize := uint64(1 << 22)
	encoder.MaxChunkSize = maxChunkSize
	params.OverrideBeaconNetworkConfig(c)
	_, err := e.EncodeWithMaxLength(buf, st.ToProtoUnsafe().(*ethpb.BeaconState))
	require.NoError(t, err)
	// Max snappy block size
	if buf.Len() <= 76490 {
		t.Errorf("buffer smaller than expected, wanted > %d but got %d", 76490, buf.Len())
	}
	decoded := new(ethpb.BeaconState)
	err = e.DecodeWithMaxLength(buf, decoded)
	assert.NoError(t, err)
}
func TestSszNetworkEncoder_NegativeMaxLength(t *testing.T) {
	e := &encoder.SszNetworkEncoder{}
	length, err := e.MaxLength(0xfffffffffff)

	assert.Equal(t, 0, length, "Received non zero length on bad message length")
	assert.ErrorContains(t, "max encoded length is negative", err)
}

func TestSszNetworkEncoder_MaxInt64(t *testing.T) {
	e := &encoder.SszNetworkEncoder{}
	length, err := e.MaxLength(math.MaxInt64 + 1)

	assert.Equal(t, 0, length, "Received non zero length on bad message length")
	assert.ErrorContains(t, "invalid length provided", err)
}

func TestSszNetworkEncoder_DecodeWithBadSnappyStream(t *testing.T) {
	st := newBadSnappyStream()
	e := &encoder.SszNetworkEncoder{}
	decoded := new(ethpb.Fork)
	err := e.DecodeWithMaxLength(st, decoded)
	assert.ErrorContains(t, io.EOF.Error(), err)
}

type badSnappyStream struct {
	varint []byte
	header []byte
	repeat []byte
	i      int
	// count how many times it was read
	counter int
	// count bytes read so far
	total int
}

func newBadSnappyStream() *badSnappyStream {
	const (
		magicBody  = "sNaPpY"
		magicChunk = "\xff\x06\x00\x00" + magicBody
	)

	header := make([]byte, len(magicChunk))
	// magicChunk == chunkTypeStreamIdentifier byte ++ 3 byte little endian len(magic body) ++ 6 byte magic body

	// header is a special chunk type, with small fixed length, to add some magic to claim it's really snappy.
	copy(header, magicChunk) // snappy library constants help us construct the common header chunk easily.

	payload := make([]byte, 4)

	// byte 0 is chunk type
	// Exploit any fancy ignored chunk type
	//   Section 4.4 Padding (chunk type 0xfe).
	//   Section 4.6. Reserved skippable chunks (chunk types 0x80-0xfd).
	payload[0] = 0xfe

	// byte 1,2,3 are chunk length (little endian)
	payload[1] = 0
	payload[2] = 0
	payload[3] = 0

	return &badSnappyStream{
		varint:  gogo.EncodeVarint(1000),
		header:  header,
		repeat:  payload,
		i:       0,
		counter: 0,
		total:   0,
	}
}

func (b *badSnappyStream) Read(p []byte) (n int, err error) {
	// Stream out varint bytes first to make test happy.
	if len(b.varint) > 0 {
		copy(p, b.varint[:1])
		b.varint = b.varint[1:]
		return 1, nil
	}
	defer func() {
		b.counter += 1
		b.total += n
	}()
	if len(b.repeat) == 0 {
		panic("no bytes to repeat")
	}
	if len(b.header) > 0 {
		n = copy(p, b.header)
		b.header = b.header[n:]
		return
	}
	for n < len(p) {
		n += copy(p[n:], b.repeat[b.i:])
		b.i = (b.i + n) % len(b.repeat)
	}
	return
}
