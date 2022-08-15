package helpers

import (
	"bytes"
	"context"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/api/grpc"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidateSync checks whether the node is currently syncing and returns an error if it is.
// It also appends syncing info to gRPC headers.
func ValidateSync(
	ctx context.Context,
	syncChecker sync.Checker,
	headFetcher blockchain.HeadFetcher,
	timeFetcher blockchain.TimeFetcher,
	optimisticModeFetcher blockchain.OptimisticModeFetcher,
) error {
	if !syncChecker.Syncing() {
		return nil
	}
	headSlot := headFetcher.HeadSlot()
	isOptimistic, err := optimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not check optimistic status: %v", err)
	}

	syncDetailsContainer := &syncDetailsContainer{
		SyncDetails: &SyncDetailsJson{
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
func IsOptimistic(ctx context.Context, st state.BeaconState, optimisticModeFetcher blockchain.OptimisticModeFetcher) (bool, error) {
	header := st.LatestBlockHeader()
	// This happens when the block at the state's slot is not missing.
	if bytes.Equal(header.StateRoot, params.BeaconConfig().ZeroHash[:]) {
		root, err := st.HashTreeRoot(ctx)
		if err != nil {
			return false, errors.Wrap(err, "could not get state root")
		}
		header.StateRoot = root[:]
	}
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

// SyncDetailsJson contains information about node sync status.
type SyncDetailsJson struct {
	HeadSlot     string `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
	IsOptimistic bool   `json:"is_optimistic"`
}

// SyncDetailsContainer is a wrapper for SyncDetails.
type syncDetailsContainer struct {
	SyncDetails *SyncDetailsJson `json:"sync_details"`
}
