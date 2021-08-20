package beacon

import (
	"context"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	corehelpers "github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListSyncCommittees retrieves the sync committees for the given epoch.
// If the epoch is not passed in, then the sync committees for the epoch of the state will be obtained.
func (bs *Server) ListSyncCommittees(ctx context.Context, req *eth.StateSyncCommitteesRequest) (*eth.StateSyncCommitteesResponse, error) {
	var st state.BeaconState
	if req.Epoch != nil {
		slot, err := corehelpers.StartSlot(*req.Epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not calculate start slot for epoch %d: %v", *req.Epoch, err)
		}
		st, err = bs.StateFetcher.State(ctx, []byte(strconv.FormatUint(uint64(slot), 10)))
		if err != nil {
			return nil, helpers.PrepareStateFetchGRPCError(err)
		}
	} else {
		var err error
		st, err = bs.StateFetcher.State(ctx, req.StateId)
		if err != nil {
			return nil, helpers.PrepareStateFetchGRPCError(err)
		}
	}

	committee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync committee: %v", err)
	}

	committeeIndices := make([]types.ValidatorIndex, len(committee.Pubkeys))
	for i, key := range committee.Pubkeys {
		index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(key))
		if !ok {
			return nil, status.Errorf(codes.Internal, "Validator index not found for pubkey %#x", bytesutil.Trunc(key))
		}
		committeeIndices[i] = index
	}

	subcommitteeCount := params.BeaconConfig().SyncCommitteeSubnetCount
	subcommittees := make([]*eth.SyncSubcommitteeValidators, subcommitteeCount)
	for i := uint64(0); i < subcommitteeCount; i++ {
		pubkeys, err := altair.SyncSubCommitteePubkeys(committee, types.CommitteeIndex(i))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get subcommittee pubkeys: %v", err)
		}
		subcommittee := &eth.SyncSubcommitteeValidators{Validators: make([]types.ValidatorIndex, len(pubkeys))}
		for j, key := range pubkeys {
			index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(key))
			if !ok {
				return nil, status.Errorf(codes.Internal, "Validator index not found for pubkey %#x", bytesutil.Trunc(key))
			}
			subcommittee.Validators[j] = index
		}
		subcommittees[i] = subcommittee
	}

	return &eth.StateSyncCommitteesResponse{
		Data: &eth.SyncCommitteeValidators{
			Validators:          committeeIndices,
			ValidatorAggregates: subcommittees,
		},
	}, nil
}

func (bs *Server) SubmitPoolSyncCommitteeSignatures(ctx context.Context, req *eth.SubmitPoolSyncCommitteeSignatures) (*empty.Empty, error) {
	panic("implement me")
}
