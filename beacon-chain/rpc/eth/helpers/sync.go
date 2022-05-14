package helpers

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/grpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidateSync checks whether the node is currently syncing and returns an error if it is.
// It also appends syncing info to gRPC headers.
func ValidateSync(ctx context.Context, syncChecker sync.Checker, headFetcher blockchain.HeadFetcher, timeFetcher blockchain.TimeFetcher) error {
	if !syncChecker.Syncing() {
		return nil
	}
	headSlot := headFetcher.HeadSlot()
	// QUESTION: should we +/ how do we pass beaconstate / modeFetcher into IsOptimistic?
	syncDetailsContainer := &SyncDetailsContainer{
		SyncDetails: &SyncDetails{
			HeadSlot:     strconv.FormatUint(uint64(headSlot), 10),
			SyncDistance: strconv.FormatUint(uint64(timeFetcher.CurrentSlot()-headSlot), 10),
			IsSyncing:    true,
			IsOptimistic: IsOptimistic(ctx) 
		},
	}
	err := grpc.AppendCustomErrorHeader(ctx, syncDetailsContainer)
	if err != nil {
		return status.Errorf(
			codes.Internal,
			"Syncing to latest head, not ready to respond. Could not prepare sync details: %v",
			err,
		)
	}
	return status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
}

// IsOptimistic checks whether the latest block header of the passed in beacon state is the header of an optimistic block.
func IsOptimistic(ctx context.Context, st state.BeaconState, optimisticSyncFetcher blockchain.OptimisticModeFetcher) (bool, error) {
	root, err := st.HashTreeRoot(ctx)
	if err != nil {
		return false, errors.Wrap(err, "could not get state root")
	}
	header := st.LatestBlockHeader()
	header.StateRoot = root[:]
	headRoot, err := header.HashTreeRoot()
	if err != nil {
		return false, errors.Wrap(err, "could not get header root")
	}
	isOptimistic, err := optimisticSyncFetcher.IsOptimisticForRoot(ctx, headRoot)
	if err != nil {
		return false, errors.Wrap(err, "could not check if block is optimistic")
	}
	return isOptimistic, nil
}
