package beacon

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListSyncCommittees retrieves the sync committees for the given epoch.
// If the epoch is not passed in, then the sync committees for the epoch of the state will be obtained.
func (bs *Server) ListSyncCommittees(ctx context.Context, req *ethpbv2.StateSyncCommitteesRequest) (*ethpbv2.StateSyncCommitteesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListSyncCommittees")
	defer span.End()

	currentSlot := bs.GenesisTimeFetcher.CurrentSlot()
	currentEpoch := slots.ToEpoch(currentSlot)
	currentPeriodStartEpoch, err := slots.SyncCommitteePeriodStartEpoch(currentEpoch)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not calculate start period for slot %d: %v",
			currentSlot,
			err,
		)
	}

	requestNextCommittee := false
	if req.Epoch != nil {
		reqPeriodStartEpoch, err := slots.SyncCommitteePeriodStartEpoch(*req.Epoch)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not calculate start period for epoch %d: %v",
				*req.Epoch,
				err,
			)
		}
		if reqPeriodStartEpoch > currentPeriodStartEpoch+params.BeaconConfig().EpochsPerSyncCommitteePeriod {
			return nil, status.Errorf(
				codes.Internal,
				"Could not fetch sync committee too far in the future. Requested epoch: %d, current epoch: %d",
				*req.Epoch, currentEpoch,
			)
		}
		if reqPeriodStartEpoch > currentPeriodStartEpoch {
			requestNextCommittee = true
			req.Epoch = &currentPeriodStartEpoch
		}
	}

	st, err := bs.stateFromRequest(ctx, &stateRequest{
		epoch:   req.Epoch,
		stateId: req.StateId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not fetch beacon state using request: %v", err)
	}

	var committeeIndices []primitives.ValidatorIndex
	var committee *ethpbalpha.SyncCommittee
	if requestNextCommittee {
		// Get the next sync committee and sync committee indices from the state.
		committeeIndices, committee, err = nextCommitteeIndicesFromState(st)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get next sync committee indices: %v", err)
		}
	} else {
		// Get the current sync committee and sync committee indices from the state.
		committeeIndices, committee, err = currentCommitteeIndicesFromState(st)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get current sync committee indices: %v", err)
		}
	}
	subcommittees, err := extractSyncSubcommittees(st, committee)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not extract sync subcommittees: %v", err)
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	return &ethpbv2.StateSyncCommitteesResponse{
		Data: &ethpbv2.SyncCommitteeValidators{
			Validators:          committeeIndices,
			ValidatorAggregates: subcommittees,
		},
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
	}, nil
}

func committeeIndicesFromState(st state.BeaconState, committee *ethpbalpha.SyncCommittee) ([]primitives.ValidatorIndex, *ethpbalpha.SyncCommittee, error) {
	committeeIndices := make([]primitives.ValidatorIndex, len(committee.Pubkeys))
	for i, key := range committee.Pubkeys {
		index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(key))
		if !ok {
			return nil, nil, fmt.Errorf(
				"validator index not found for pubkey %#x",
				bytesutil.Trunc(key),
			)
		}
		committeeIndices[i] = index
	}
	return committeeIndices, committee, nil
}

func currentCommitteeIndicesFromState(st state.BeaconState) ([]primitives.ValidatorIndex, *ethpbalpha.SyncCommittee, error) {
	committee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, nil, fmt.Errorf(
			"could not get sync committee: %v", err,
		)
	}

	return committeeIndicesFromState(st, committee)
}

func nextCommitteeIndicesFromState(st state.BeaconState) ([]primitives.ValidatorIndex, *ethpbalpha.SyncCommittee, error) {
	committee, err := st.NextSyncCommittee()
	if err != nil {
		return nil, nil, fmt.Errorf(
			"could not get sync committee: %v", err,
		)
	}

	return committeeIndicesFromState(st, committee)
}

func extractSyncSubcommittees(st state.BeaconState, committee *ethpbalpha.SyncCommittee) ([]*ethpbv2.SyncSubcommitteeValidators, error) {
	subcommitteeCount := params.BeaconConfig().SyncCommitteeSubnetCount
	subcommittees := make([]*ethpbv2.SyncSubcommitteeValidators, subcommitteeCount)
	for i := uint64(0); i < subcommitteeCount; i++ {
		pubkeys, err := altair.SyncSubCommitteePubkeys(committee, primitives.CommitteeIndex(i))
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get subcommittee pubkeys: %v", err,
			)
		}
		subcommittee := &ethpbv2.SyncSubcommitteeValidators{Validators: make([]primitives.ValidatorIndex, len(pubkeys))}
		for j, key := range pubkeys {
			index, ok := st.ValidatorIndexByPubkey(bytesutil.ToBytes48(key))
			if !ok {
				return nil, fmt.Errorf(
					"validator index not found for pubkey %#x",
					bytesutil.Trunc(key),
				)
			}
			subcommittee.Validators[j] = index
		}
		subcommittees[i] = subcommittee
	}
	return subcommittees, nil
}
