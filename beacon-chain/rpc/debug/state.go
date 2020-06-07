package debug

import (
	"context"
	"sort"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBeaconState retrieves an ssz-encoded beacon state
// from the beacon node by either a slot or block root.
func (ds *Server) GetBeaconState(
	ctx context.Context,
	req *pbrpc.BeaconStateRequest,
) (*pbrpc.SSZResponse, error) {
	if !featureconfig.Get().NewStateMgmt {
		return nil, status.Error(codes.FailedPrecondition, "Requires --enable-new-state-mgmt to function")
	}

	switch q := req.QueryFilter.(type) {
	case *pbrpc.BeaconStateRequest_Slot:
		currentSlot := ds.GenesisTimeFetcher.CurrentSlot()
		requestedSlot := q.Slot
		if requestedSlot > currentSlot {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"Cannot retrieve information about a slot in the future, current slot %d, requested slot %d",
				currentSlot,
				requestedSlot,
			)
		}

		st, err := ds.StateGen.StateBySlot(ctx, q.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not compute state by slot: %v", err)
		}
		encoded, err := st.CloneInnerState().MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not ssz encode beacon state: %v", err)
		}
		return &pbrpc.SSZResponse{
			Encoded: encoded,
		}, nil
	case *pbrpc.BeaconStateRequest_BlockRoot:
		st, err := ds.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(q.BlockRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not compute state by block root: %v", err)
		}
		encoded, err := st.CloneInnerState().MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not ssz encode beacon state: %v", err)
		}
		return &pbrpc.SSZResponse{
			Encoded: encoded,
		}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "Need to specify either a block root or slot to request state")
	}
}

// GetIndividualVotes retrieves individual voting status of validators.
func (ds *Server) GetIndividualVotes(
	ctx context.Context,
	req *pbrpc.IndividualVotesRequest,
) (*pbrpc.IndividualVotesRespond, error) {
	if !featureconfig.Get().NewStateMgmt {
		return nil, status.Error(codes.FailedPrecondition, "Requires --enable-new-state-mgmt to function")
	}

	currentEpoch := helpers.SlotToEpoch(ds.GenesisTimeFetcher.CurrentSlot())
	if req.Epoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			currentEpoch,
			req.Epoch,
		)
	}

	requestedState, err := ds.StateGen.StateBySlot(ctx, helpers.StartSlot(req.Epoch))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve archived state for epoch %d: %v", req.Epoch, err)
	}
	// Track filtered validators to prevent duplication in the response.
	filtered := map[uint64]bool{}
	filteredIndices := make([]uint64, 0)
	votes := make([]*pbrpc.IndividualVotesRespond_IndividualVote, 0, len(req.Indices)+len(req.PublicKeys))
	// Filter out assignments by public keys.
	for _, pubKey := range req.PublicKeys {
		index, ok := requestedState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
		if !ok {
			votes = append(votes, &pbrpc.IndividualVotesRespond_IndividualVote{PublicKey: pubKey, ValidatorIndex: ^uint64(0)})
		}
		filtered[index] = true
		filteredIndices = append(filteredIndices, index)
	}
	// Filter out assignments by validator indices.
	for _, index := range req.Indices {
		if !filtered[index] {
			filteredIndices = append(filteredIndices, index)
		}
	}
	sort.Slice(filteredIndices, func(i, j int) bool {
		return filteredIndices[i] < filteredIndices[j]
	})

	v, b, err := precompute.New(ctx, requestedState)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not set up pre compute instance")
	}
	v, b, err = precompute.ProcessAttestations(ctx, requestedState, v, b)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not pre compute attestations")
	}

	validators := requestedState.ValidatorsReadOnly()
	for _, index := range filteredIndices {
		if int(index) >= len(v) {
			continue
		}
		pb := validators[index].PublicKey()
		votes = append(votes, &pbrpc.IndividualVotesRespond_IndividualVote{
			Epoch:                            req.Epoch,
			PublicKey:                        pb[:],
			ValidatorIndex:                   index,
			IsSlashed:                        v[index].IsSlashed,
			IsWithdrawableInCurrentEpoch:     v[index].IsWithdrawableCurrentEpoch,
			IsActiveInCurrentEpoch:           v[index].IsActiveCurrentEpoch,
			IsActiveInPreviousEpoch:          v[index].IsActivePrevEpoch,
			IsCurrentEpochAttester:           v[index].IsCurrentEpochAttester,
			IsCurrentEpochTargetAttester:     v[index].IsCurrentEpochTargetAttester,
			IsPreviousEpochAttester:          v[index].IsPrevEpochAttester,
			IsPreviousEpochTargetAttester:    v[index].IsPrevEpochTargetAttester,
			IsPreviousEpochHeadAttester:      v[index].IsPrevEpochHeadAttester,
			CurrentEpochEffectiveBalanceGwei: v[index].CurrentEpochEffectiveBalance,
			InclusionSlot:                    v[index].InclusionSlot,
			InclusionDistance:                v[index].InclusionDistance,
		})
	}

	return &pbrpc.IndividualVotesRespond{
		IndividualVotes: votes,
	}, nil
}
