package iface

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type BeaconChainClient interface {
	ListAttestations(ctx context.Context, in *ethpb.ListAttestationsRequest) (*ethpb.ListAttestationsResponse, error)
	ListIndexedAttestations(ctx context.Context, in *ethpb.ListIndexedAttestationsRequest) (*ethpb.ListIndexedAttestationsResponse, error)
	// Deprecated: Do not use.
	StreamAttestations(ctx context.Context, in *empty.Empty) (ethpb.BeaconChain_StreamAttestationsClient, error)
	// Deprecated: Do not use.
	StreamIndexedAttestations(ctx context.Context, in *empty.Empty) (ethpb.BeaconChain_StreamIndexedAttestationsClient, error)
	AttestationPool(ctx context.Context, in *ethpb.AttestationPoolRequest) (*ethpb.AttestationPoolResponse, error)
	ListBeaconBlocks(ctx context.Context, in *ethpb.ListBlocksRequest) (*ethpb.ListBeaconBlocksResponse, error)
	// Deprecated: Do not use.
	StreamBlocks(ctx context.Context, in *ethpb.StreamBlocksRequest) (ethpb.BeaconChain_StreamBlocksClient, error)
	// Deprecated: Do not use.
	StreamChainHead(ctx context.Context, in *empty.Empty) (ethpb.BeaconChain_StreamChainHeadClient, error)
	GetChainHead(ctx context.Context, in *empty.Empty) (*ethpb.ChainHead, error)
	ListBeaconCommittees(ctx context.Context, in *ethpb.ListCommitteesRequest) (*ethpb.BeaconCommittees, error)
	ListValidatorBalances(ctx context.Context, in *ethpb.ListValidatorBalancesRequest) (*ethpb.ValidatorBalances, error)
	ListValidators(ctx context.Context, in *ethpb.ListValidatorsRequest) (*ethpb.Validators, error)
	GetValidator(ctx context.Context, in *ethpb.GetValidatorRequest) (*ethpb.Validator, error)
	GetValidatorActiveSetChanges(ctx context.Context, in *ethpb.GetValidatorActiveSetChangesRequest) (*ethpb.ActiveSetChanges, error)
	GetValidatorQueue(ctx context.Context, in *empty.Empty) (*ethpb.ValidatorQueue, error)
	GetValidatorPerformance(ctx context.Context, in *ethpb.ValidatorPerformanceRequest) (*ethpb.ValidatorPerformanceResponse, error)
	ListValidatorAssignments(ctx context.Context, in *ethpb.ListValidatorAssignmentsRequest) (*ethpb.ValidatorAssignments, error)
	GetValidatorParticipation(ctx context.Context, in *ethpb.GetValidatorParticipationRequest) (*ethpb.ValidatorParticipationResponse, error)
	GetBeaconConfig(ctx context.Context, in *empty.Empty) (*ethpb.BeaconConfig, error)
	// Deprecated: Do not use.
	StreamValidatorsInfo(ctx context.Context) (ethpb.BeaconChain_StreamValidatorsInfoClient, error)
	SubmitAttesterSlashing(ctx context.Context, in *ethpb.AttesterSlashing) (*ethpb.SubmitSlashingResponse, error)
	SubmitProposerSlashing(ctx context.Context, in *ethpb.ProposerSlashing) (*ethpb.SubmitSlashingResponse, error)
	GetIndividualVotes(ctx context.Context, in *ethpb.IndividualVotesRequest) (*ethpb.IndividualVotesRespond, error)
}
