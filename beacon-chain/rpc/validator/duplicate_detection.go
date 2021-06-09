package validator

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"go.opencensus.io/trace"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Number of Epochs to check
var NoEpochsToChech uint64

// ADD COMMENTS
func (vs *Server) DetectDoppelganger(ctx context.Context, req *ethpb.DetectDoppelgangerRequest) (*ethpb.DetectDoppelgangerResponse, error) {
	log.Info("Doppelganger rpc service started")
	// Head state
	head, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Doppelganger rpc service - Could not get head state: %v", err)
	}
	// Head epoch
	headEpoch := types.Epoch(head.Slot() / params.BeaconConfig().SlotsPerEpoch)
	ctx, span := trace.StartSpan(ctx, "Doppelganger rpc")
	defer span.End()

	NoEpochsToChech = params.BeaconConfig().DuplicateValidatorEpochsCheck

	baseRewardFactor := params.BeaconConfig().BaseRewardFactor
	baseRewardsPerEpoch := params.BeaconConfig().BaseRewardsPerEpoch

	for _, pkt := range req.PubKeysTargets {
		if headEpoch-pkt.TargetEpoch < 2 {
			continue
		}

		// Ensure the balance has been increasing due to non-activity
		ok := true
		headState := head
		for i := NoEpochsToChech; i > 0 && ok; i++ {
			// Calculate base_reward at head-i
			// Get balance at head-i+1 and head-i
			prevStateRoot, err := headState.HashTreeRoot(ctx)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Doppelganger rpc service - Could not get previous "+
					"state: %v", err)
			}
			prevState, err := vs.StateGen.StateByRoot(ctx, prevStateRoot)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Doppelganger rpc service - Could not get "+
					"previous heat state: %v", err)
			}
			totalBalance, err := helpers.TotalActiveBalance(prevState)
			if err != nil {
				return  &ethpb.DetectDoppelgangerResponse{PublicKey: nil,
					DuplicateFound: false}, status.Errorf(codes.Internal, "Doppelganger rpc service - Could not get calculate balance: %v", err)
			}
			if totalBalance == 0 {
				totalBalance = 1
			}
			balanceSqrt := mathutil.IntegerSquareRoot(totalBalance)

			valIdx, err := vs.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: pkt.PubKey})
			if err != nil {
				return  &ethpb.DetectDoppelgangerResponse{PublicKey: nil,
					DuplicateFound: false}, status.Errorf(codes.Internal, "Doppelganger rpc service - : %v", err)
			}
			valPrevBalance, err := prevState.BalanceAtIndex(valIdx.Index)
			if err != nil {
				return  &ethpb.DetectDoppelgangerResponse{PublicKey: nil,
					DuplicateFound: false}, status.Errorf(codes.Internal, "Doppelganger rpc service - could not get balance: %v", err)
			}
			valHeadBalance, err := headState.BalanceAtIndex(valIdx.Index)
			if err != nil {
				return  &ethpb.DetectDoppelgangerResponse{PublicKey: nil,
					DuplicateFound: false}, status.Errorf(codes.Internal, "Doppelganger rpc service - could not get balance: %v", err)
			}
			base_reward := valPrevBalance * baseRewardFactor / balanceSqrt / baseRewardsPerEpoch

			if valHeadBalance-valPrevBalance > 2*base_reward {
				return &ethpb.DetectDoppelgangerResponse{PublicKey: pkt.PubKey,
					DuplicateFound: true}, nil
			}
			headState = prevState
		}

	}
	// see reward_penalty.go precompute
	// see attestor.go
	return &ethpb.DetectDoppelgangerResponse{PublicKey: nil,
		DuplicateFound: false}, nil
}
