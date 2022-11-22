package beacon

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
)

// GetLightClientBootstrap - implements https://github.com/ethereum/beacon-APIs/blob/263f4ed6c263c967f13279c7a9f5629b51c5fc55/apis/beacon/light_client/bootstrap.yaml
func (bs *Server) GetLightClientBootstrap(ctx context.Context, req *ethpbv2.LightClientBootstrapRequest) (*ethpbv2.LightClientBootstrapResponse, error) {
	// Prepare
	ctx, span := trace.StartSpan(ctx, "beacon.GetLightClientBootstrap")
	defer span.End()

	// Get the block
	var blockRoot [32]byte
	copy(blockRoot[:], req.BlockRoot)

	blk, err := bs.BeaconDB.Block(ctx, blockRoot)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, err
	}

	// Get the state
	state, err := bs.StateFetcher.StateBySlot(ctx, blk.Block().Slot())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state by slot: %v", err)
	}

	bootstrap, err := helpers.CreateLightClientBootstrap(ctx, state)
	if err != nil {
		return nil, err
	}

	result := &ethpbv2.LightClientBootstrapResponse{
		Version: ethpbv2.Version(blk.Version()),
		Data:    bootstrap,
	}

	return result, nil
}

// GetLightClientUpdatesByRange - implements https://github.com/ethereum/beacon-APIs/blob/263f4ed6c263c967f13279c7a9f5629b51c5fc55/apis/beacon/light_client/updates.yaml
func (bs *Server) GetLightClientUpdatesByRange(ctx context.Context, req *ethpbv2.LightClientUpdatesByRangeRequest) (*ethpbv2.LightClientUpdatesByRangeResponse, error) {
	// Prepare
	ctx, span := trace.StartSpan(ctx, "beacon.GetLightClientUpdatesByRange")
	defer span.End()

	// Determine slots per period
	config := params.BeaconConfig()
	slotsPerPeriod := uint64(config.EpochsPerSyncCommitteePeriod) * uint64(config.SlotsPerEpoch)

	// Adjust count based on configuration
	count := uint64(req.Count)
	if count > config.MaxRequestLightClientUpdates {
		count = config.MaxRequestLightClientUpdates
	}

	// Determine the start and end periods
	startPeriod := req.StartPeriod
	endPeriod := startPeriod + count - 1

	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	lHeadSlot := uint64(headState.Slot())
	headPeriod := lHeadSlot / slotsPerPeriod
	if headPeriod < endPeriod {
		endPeriod = headPeriod
	}

	// Populate updates
	var updates []*ethpbv2.LightClientUpdateWithVersion
	for period := startPeriod; period <= endPeriod; period++ {
		// Get the last known state of the period,
		//    1. We wish the block has a parent in the same period if possible
		//	  2. We wish the block has a state in the same period
		lLastSlotInPeriod := period*slotsPerPeriod + slotsPerPeriod - 1
		if lLastSlotInPeriod > lHeadSlot {
			lLastSlotInPeriod = lHeadSlot
		}
		lFirstSlotInPeriod := period * slotsPerPeriod

		var state state.BeaconState
		for lSlot := lLastSlotInPeriod; lSlot >= lFirstSlotInPeriod; lSlot-- {
			state, err = bs.StateFetcher.StateBySlot(ctx, types.Slot(lSlot))
			if err == nil {
				break
			}
		}

		if state == nil {
			// No valid state found for the period
			continue
		}

		// Get the block
		slot := state.Slot()
		blocks, err := bs.BeaconDB.BlocksBySlot(ctx, slot)
		if err != nil || len(blocks) == 0 {
			continue
		}
		block := blocks[0]

		// Get attested state
		attestedRoot := block.Block().ParentRoot()
		attestedBlock, err := bs.BeaconDB.Block(ctx, attestedRoot)
		if err != nil || attestedBlock == nil {
			continue
		}

		attestedSlot := attestedBlock.Block().Slot()
		attestedState, err := bs.StateFetcher.StateBySlot(ctx, attestedSlot)
		if err != nil {
			continue
		}

		// Get finalized block
		var finalizedBlock interfaces.SignedBeaconBlock
		finalizedCheckPoint := attestedState.FinalizedCheckpoint()
		if finalizedCheckPoint != nil {
			finalizedRoot := bytesutil.ToBytes32(finalizedCheckPoint.Root)
			finalizedBlock, err = bs.BeaconDB.Block(ctx, finalizedRoot)
			if err != nil {
				finalizedBlock = nil
			}
		}

		update, err := helpers.CreateLightClientUpdate(
			ctx,
			config,
			slotsPerPeriod,
			state,
			block,
			attestedState,
			finalizedBlock,
		)

		if err == nil {
			updates = append(updates, &ethpbv2.LightClientUpdateWithVersion{
				Version: ethpbv2.Version(attestedState.Version()),
				Data:    update,
			})
		}
	}

	if len(updates) == 0 {
		return nil, status.Errorf(codes.NotFound, "No updates found")
	}

	result := ethpbv2.LightClientUpdatesByRangeResponse{
		Updates: updates,
	}

	return &result, nil
}

// GetLightClientFinalityUpdate - implements https://github.com/ethereum/beacon-APIs/blob/263f4ed6c263c967f13279c7a9f5629b51c5fc55/apis/beacon/light_client/finality_update.yaml
func (bs *Server) GetLightClientFinalityUpdate(ctx context.Context, _ *empty.Empty) (*ethpbv2.LightClientFinalityUpdateResponse, error) {
	// Prepare
	ctx, span := trace.StartSpan(ctx, "beacon.GetLightClientFinalityUpdate")
	defer span.End()

	// Determine slots per period
	config := params.BeaconConfig()
	slotsPerPeriod := uint64(config.EpochsPerSyncCommitteePeriod) * uint64(config.SlotsPerEpoch)

	// Get the current state
	state, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Get the block
	latestBlockHeader := state.LatestBlockHeader()
	latestBlockHeaderRoot, err := latestBlockHeader.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get latest block header root: %v", err)
	}

	block, err := bs.BeaconDB.Block(ctx, latestBlockHeaderRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get latest block: %v", err)
	}

	// Get attested state
	attestedRoot := block.Block().ParentRoot()
	attestedBlock, err := bs.BeaconDB.Block(ctx, attestedRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get attested block: %v", err)
	}

	attestedSlot := attestedBlock.Block().Slot()
	attestedState, err := bs.StateFetcher.StateBySlot(ctx, attestedSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get attested state: %v", err)
	}

	// Get finalized block
	var finalizedBlock interfaces.SignedBeaconBlock
	finalizedCheckPoint := attestedState.FinalizedCheckpoint()
	if finalizedCheckPoint != nil {
		finalizedRoot := bytesutil.ToBytes32(finalizedCheckPoint.Root)
		finalizedBlock, err = bs.BeaconDB.Block(ctx, finalizedRoot)
		if err != nil {
			finalizedBlock = nil
		}
	}

	update, err := helpers.CreateLightClientUpdate(
		ctx,
		config,
		slotsPerPeriod,
		state,
		block,
		attestedState,
		finalizedBlock,
	)

	if err != nil {
		return nil, err
	}

	finalityUpdate := helpers.CreateLightClientFinalityUpdate(update)

	// Return the result
	result := &ethpbv2.LightClientFinalityUpdateResponse{
		Version: ethpbv2.Version(attestedState.Version()),
		Data:    finalityUpdate,
	}

	return result, nil
}

// GetLightClientOptimisticUpdate - implements https://github.com/ethereum/beacon-APIs/blob/263f4ed6c263c967f13279c7a9f5629b51c5fc55/apis/beacon/light_client/optimistic_update.yaml
func (bs *Server) GetLightClientOptimisticUpdate(ctx context.Context, _ *empty.Empty) (*ethpbv2.LightClientOptimisticUpdateResponse, error) {
	// Prepare
	ctx, span := trace.StartSpan(ctx, "beacon.GetLightClientOptimisticUpdate")
	defer span.End()

	// Determine slots per period
	config := params.BeaconConfig()
	slotsPerPeriod := uint64(config.EpochsPerSyncCommitteePeriod) * uint64(config.SlotsPerEpoch)

	// Get the current state
	state, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Get the block
	latestBlockHeader := state.LatestBlockHeader()
	latestBlockHeaderRoot, err := latestBlockHeader.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get latest block header root: %v", err)
	}

	block, err := bs.BeaconDB.Block(ctx, latestBlockHeaderRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get latest block: %v", err)
	}

	// Get attested state
	attestedRoot := block.Block().ParentRoot()
	attestedBlock, err := bs.BeaconDB.Block(ctx, attestedRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get attested block: %v", err)
	}

	attestedSlot := attestedBlock.Block().Slot()
	attestedState, err := bs.StateFetcher.StateBySlot(ctx, attestedSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get attested state: %v", err)
	}

	// Get finalized block
	var finalizedBlock interfaces.SignedBeaconBlock
	finalizedCheckPoint := attestedState.FinalizedCheckpoint()
	if finalizedCheckPoint != nil {
		finalizedRoot := bytesutil.ToBytes32(finalizedCheckPoint.Root)
		finalizedBlock, err = bs.BeaconDB.Block(ctx, finalizedRoot)
		if err != nil {
			finalizedBlock = nil
		}
	}

	update, err := helpers.CreateLightClientUpdate(
		ctx,
		config,
		slotsPerPeriod,
		state,
		block,
		attestedState,
		finalizedBlock,
	)

	if err != nil {
		return nil, err
	}

	optimisticUpdate := helpers.CreateLightClientOptimisticUpdate(update)

	// Return the result
	result := &ethpbv2.LightClientOptimisticUpdateResponse{
		Version: ethpbv2.Version(attestedState.Version()),
		Data:    optimisticUpdate,
	}

	return result, nil
}
