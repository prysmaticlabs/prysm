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
func ValidateSync(ctx context.Context, syncChecker sync.Checker, headFetcher blockchain.HeadFetcher, timeFetcher blockchain.TimeFetcher, optimisticModeFetcher blockchain.OptimisticModeFetcher) error {
	if !syncChecker.Syncing() {
		return nil
	}
	headSlot := headFetcher.HeadSlot()

	headState, err := headFetcher.HeadState(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	isOptimistic, err := IsOptimistic(ctx, headState, optimisticModeFetcher) // opModeFetcher comes from param
	if err != nil {
		return status.Errorf(codes.Internal, "Could not check optimistic status: %v", err)
	}

	// QUESTION: How do I import the structs from /rpc/apimiddleware/structs.go here without importing the whole middleware package?
	syncDetailsContainer := &syncDetailsContainer{
		SyncDetails: &syncDetailsJson{
			HeadSlot:     strconv.FormatUint(uint64(headSlot), 10),
			SyncDistance: strconv.FormatUint(uint64(timeFetcher.CurrentSlot()-headSlot), 10),
			IsSyncing:    true,
			IsOptimistic: isOptimistic,
		},
	}
	err = grpc.AppendCustomErrorHeader(ctx, syncDetailsContainer)
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
// This is exposed to end-users who interpret `true` as "your Prysm beacon node is optimistically tracking head - your execution node isn't yet fully synced"
func IsOptimistic(ctx context.Context, st state.BeaconState, optimisticModeFetcher blockchain.OptimisticModeFetcher) (bool, error) {
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

	isOptimistic, err := optimisticModeFetcher.IsOptimisticForRoot(ctx, headRoot)
	if err != nil {
		return false, errors.Wrap(err, "could not check if block is optimistic")
	}

	return isOptimistic, nil
}
