package beacon

import (
	"context"
	"sort"

	"github.com/golang/protobuf/ptypes/empty"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/statefetcher"
	"github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (bs *Server) ListSyncCommittees(ctx context.Context, req *eth.StateSyncCommitteesRequest) (*eth.StateSyncCommitteesResponse, error) {
	state, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		if stateNotFoundErr, ok := err.(*statefetcher.StateNotFoundError); ok {
			return nil, status.Errorf(codes.NotFound, "State not found: %v", stateNotFoundErr)
		} else if parseErr, ok := err.(*statefetcher.StateIdParseError); ok {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid state ID: %v", parseErr)
		}
		return nil, status.Errorf(codes.Internal, "Invalid state ID: %v", err)
	}

	committee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get sync committee: %v", err)
	}

	committeeIndices := make([]types.ValidatorIndex, len(committee.Pubkeys))
	for i, key := range committee.Pubkeys {
		index, ok := state.ValidatorIndexByPubkey(bytesutil.ToBytes48(key))
		if !ok {
			return nil, status.Errorf(codes.Internal, "Validator index not found for pubkey %#x", bytesutil.Trunc(key))
		}
		committeeIndices[i] = index
	}
	sort.Slice(committeeIndices, func(i, j int) bool {
		return committeeIndices[i] < committeeIndices[j]
	})

	subcommitteeCount := params.BeaconConfig().SyncCommitteeSubnetCount
	subcommittees := make([]*eth.SyncSubcommittee, subcommitteeCount)
	for i := uint64(0); i < subcommitteeCount; i++ {
		pubkeys, err := altair.SyncSubCommitteePubkeys(committee, types.CommitteeIndex(i))
		subcommittee := &eth.SyncSubcommittee{Validators: make([]types.ValidatorIndex, len(pubkeys))}
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get subcommittee pubkeys: %v", err)
		}
		for j, key := range pubkeys {
			index, ok := state.ValidatorIndexByPubkey(bytesutil.ToBytes48(key))
			if !ok {
				return nil, status.Errorf(codes.Internal, "Validator index not found for pubkey %#x", bytesutil.Trunc(key))
			}
			subcommittee.Validators[j] = index
		}
		sort.Slice(subcommittee.Validators, func(i, j int) bool {
			return subcommittee.Validators[i] < subcommittee.Validators[j]
		})
		subcommittees[i] = subcommittee
	}

	return &eth.StateSyncCommitteesResponse{
		Data: &eth.SyncCommittee{
			Validators:          committeeIndices,
			ValidatorAggregates: subcommittees,
		},
	}, nil
}

func (bs *Server) SubmitSyncCommitteeSignature(ctx context.Context, message *eth.SyncCommitteeMessage) (*empty.Empty, error) {
	panic("implement me")
}
