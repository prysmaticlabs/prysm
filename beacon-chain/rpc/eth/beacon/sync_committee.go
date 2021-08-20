package beacon

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListSyncCommittees retrieves the sync committees for the given epoch.
// If the epoch is not passed in, then the sync committees for the epoch of the state will be obtained.
func (bs *Server) ListSyncCommittees(ctx context.Context, req *eth.StateSyncCommitteesRequest) (*eth.StateSyncCommitteesResponse, error) {
	st, err := bs.stateFromRequest(ctx, &stateRequest{
		epoch:   req.Epoch,
		stateId: req.StateId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not fetch beacon state using request: %v", err)
	}

	// Get the current sync committee and sync committee indices from the state.
	committeeIndices, committee, err := currentCommitteeIndicesFromState(st)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync committee indices from state: %v", err)
	}
	subcommittees, err := extractSyncSubcommittees(st, committee)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not extract sync subcommittees: %v", err)
	}

	return &eth.StateSyncCommitteesResponse{
		Data: &eth.SyncCommitteeValidators{
			Validators:          committeeIndices,
			ValidatorAggregates: subcommittees,
		},
	}, nil
}

func (bs *Server) SubmitSyncCommitteeSignature(ctx context.Context, message *eth.SyncCommitteeMessage) (*empty.Empty, error) {
	panic("implement me")
}

func currentCommitteeIndicesFromState(st state.BeaconState) ([]types.ValidatorIndex, *ethpb.SyncCommittee, error) {
	committee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, nil, status.Errorf(
			codes.Internal, "Could not get sync committee: %v", err,
		)
	}

	committeeIndices := make([]types.ValidatorIndex, len(committee.Pubkeys))
	for i, key := range committee.Pubkeys {
		index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(key))
		if !ok {
			return nil, nil, status.Errorf(
				codes.Internal,
				"Validator index not found for pubkey %#x",
				bytesutil.Trunc(key),
			)
		}
		committeeIndices[i] = index
	}
	return committeeIndices, committee, nil
}

func extractSyncSubcommittees(st state.BeaconState, committee *ethpb.SyncCommittee) ([]*eth.SyncSubcommitteeValidators, error) {
	subcommitteeCount := params.BeaconConfig().SyncCommitteeSubnetCount
	subcommittees := make([]*eth.SyncSubcommitteeValidators, subcommitteeCount)
	for i := uint64(0); i < subcommitteeCount; i++ {
		pubkeys, err := altair.SyncSubCommitteePubkeys(committee, types.CommitteeIndex(i))
		if err != nil {
			return nil, status.Errorf(
				codes.Internal, "Failed to get subcommittee pubkeys: %v", err,
			)
		}
		subcommittee := &eth.SyncSubcommitteeValidators{Validators: make([]types.ValidatorIndex, len(pubkeys))}
		for j, key := range pubkeys {
			index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(key))
			if !ok {
				return nil, status.Errorf(
					codes.Internal,
					"Validator index not found for pubkey %#x",
					bytesutil.Trunc(key),
				)
			}
			subcommittee.Validators[j] = index
		}
		subcommittees[i] = subcommittee
	}
	return subcommittees, nil
}
