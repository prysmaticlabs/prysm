//go:build use_beacon_api
// +build use_beacon_api

package beacon_api

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	iface "github.com/prysmaticlabs/prysm/v3/validator/client/iface"
)

type beaconApiValidatorClient struct {
}

func (*beaconApiValidatorClient) GetDuties(_ context.Context, _ *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.GetDuties is not implemented")
}

func (*beaconApiValidatorClient) CheckDoppelGanger(_ context.Context, _ *ethpb.DoppelGangerRequest) (*ethpb.DoppelGangerResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.CheckDoppelGanger is not implemented")
}

func (*beaconApiValidatorClient) DomainData(_ context.Context, _ *ethpb.DomainRequest) (*ethpb.DomainResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.DomainData is not implemented")
}

func (*beaconApiValidatorClient) GetAttestationData(_ context.Context, _ *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.GetAttestationData is not implemented")
}

func (*beaconApiValidatorClient) GetBeaconBlock(_ context.Context, _ *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.GetBeaconBlock is not implemented")
}

func (*beaconApiValidatorClient) GetFeeRecipientByPubKey(_ context.Context, _ *ethpb.FeeRecipientByPubKeyRequest) (*ethpb.FeeRecipientByPubKeyResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.GetFeeRecipientByPubKey is not implemented")
}

func (*beaconApiValidatorClient) GetSyncCommitteeContribution(_ context.Context, _ *ethpb.SyncCommitteeContributionRequest) (*ethpb.SyncCommitteeContribution, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.GetSyncCommitteeContribution is not implemented")
}

func (*beaconApiValidatorClient) GetSyncMessageBlockRoot(_ context.Context, _ *empty.Empty) (*ethpb.SyncMessageBlockRootResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.GetSyncMessageBlockRoot is not implemented")
}

func (*beaconApiValidatorClient) GetSyncSubcommitteeIndex(_ context.Context, _ *ethpb.SyncSubcommitteeIndexRequest) (*ethpb.SyncSubcommitteeIndexResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.GetSyncSubcommitteeIndex is not implemented")
}

func (*beaconApiValidatorClient) MultipleValidatorStatus(_ context.Context, _ *ethpb.MultipleValidatorStatusRequest) (*ethpb.MultipleValidatorStatusResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.MultipleValidatorStatus is not implemented")
}

func (*beaconApiValidatorClient) PrepareBeaconProposer(_ context.Context, _ *ethpb.PrepareBeaconProposerRequest) (*empty.Empty, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.PrepareBeaconProposer is not implemented")
}

func (*beaconApiValidatorClient) ProposeAttestation(_ context.Context, _ *ethpb.Attestation) (*ethpb.AttestResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.ProposeAttestation is not implemented")
}

func (*beaconApiValidatorClient) ProposeBeaconBlock(_ context.Context, _ *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.ProposeBeaconBlock is not implemented")
}

func (*beaconApiValidatorClient) ProposeExit(_ context.Context, _ *ethpb.SignedVoluntaryExit) (*ethpb.ProposeExitResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.ProposeExit is not implemented")
}

func (*beaconApiValidatorClient) StreamBlocksAltair(_ context.Context, _ *ethpb.StreamBlocksRequest) (ethpb.BeaconNodeValidator_StreamBlocksAltairClient, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.StreamBlocksAltair is not implemented")
}

func (*beaconApiValidatorClient) StreamDuties(_ context.Context, _ *ethpb.DutiesRequest) (ethpb.BeaconNodeValidator_StreamDutiesClient, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.StreamDuties is not implemented")
}

func (*beaconApiValidatorClient) SubmitAggregateSelectionProof(_ context.Context, _ *ethpb.AggregateSelectionRequest) (*ethpb.AggregateSelectionResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitAggregateSelectionProof is not implemented")
}

func (*beaconApiValidatorClient) SubmitSignedAggregateSelectionProof(_ context.Context, _ *ethpb.SignedAggregateSubmitRequest) (*ethpb.SignedAggregateSubmitResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitSignedAggregateSelectionProof is not implemented")
}

func (*beaconApiValidatorClient) SubmitSignedContributionAndProof(_ context.Context, _ *ethpb.SignedContributionAndProof) (*empty.Empty, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitSignedContributionAndProof is not implemented")
}

func (*beaconApiValidatorClient) SubmitSyncMessage(_ context.Context, _ *ethpb.SyncCommitteeMessage) (*empty.Empty, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitSyncMessage is not implemented")
}

func (*beaconApiValidatorClient) SubmitValidatorRegistrations(_ context.Context, _ *ethpb.SignedValidatorRegistrationsV1) (*empty.Empty, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.SubmitValidatorRegistrations is not implemented")
}

func (*beaconApiValidatorClient) SubscribeCommitteeSubnets(_ context.Context, _ *ethpb.CommitteeSubnetsSubscribeRequest) (*empty.Empty, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.SubscribeCommitteeSubnets is not implemented")
}

func (*beaconApiValidatorClient) ValidatorIndex(_ context.Context, _ *ethpb.ValidatorIndexRequest) (*ethpb.ValidatorIndexResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.ValidatorIndex is not implemented")
}

func (*beaconApiValidatorClient) ValidatorStatus(_ context.Context, _ *ethpb.ValidatorStatusRequest) (*ethpb.ValidatorStatusResponse, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.ValidatorStatus is not implemented")
}

func (*beaconApiValidatorClient) WaitForActivation(_ context.Context, _ *ethpb.ValidatorActivationRequest) (ethpb.BeaconNodeValidator_WaitForActivationClient, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.WaitForActivation is not implemented")
}

// Deprecated: Do not use.
func (*beaconApiValidatorClient) WaitForChainStart(_ context.Context, _ *empty.Empty) (ethpb.BeaconNodeValidator_WaitForChainStartClient, error) {
	// TODO: Implement me
	panic("beaconApiValidatorClient.WaitForChainStart is not implemented")
}

func NewBeaconApiValidatorClient() iface.ValidatorClient {
	return &beaconApiValidatorClient{}
}
