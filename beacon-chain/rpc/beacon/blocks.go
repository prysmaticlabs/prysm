package beacon

import (
	"context"
	"strconv"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListBlocks retrieves blocks by root, slot, or epoch.
//
// The server may return multiple blocks in the case that a slot or epoch is
// provided as the filter criteria. The server may return an empty list when
// no blocks in their database match the filter criteria. This RPC should
// not return NOT_FOUND. Only one filter criteria should be used.
func (bs *Server) ListBlocks(
	ctx context.Context, req *ethpb.ListBlocksRequest,
) (*ethpb.ListBlocksResponse, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	switch q := req.QueryFilter.(type) {
	case *ethpb.ListBlocksRequest_Epoch:
		startSlot := q.Epoch * params.BeaconConfig().SlotsPerEpoch
		endSlot := startSlot + params.BeaconConfig().SlotsPerEpoch - 1

		blks, err := bs.BeaconDB.Blocks(ctx, filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to get blocks: %v", err)
		}

		numBlks := len(blks)
		if numBlks == 0 {
			return &ethpb.ListBlocksResponse{
				BlockContainers: make([]*ethpb.BeaconBlockContainer, 0),
				TotalSize:       0,
				NextPageToken:   strconv.Itoa(0),
			}, nil
		}

		start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not paginate blocks: %v", err)
		}

		returnedBlks := blks[start:end]
		containers := make([]*ethpb.BeaconBlockContainer, len(returnedBlks))
		for i, b := range returnedBlks {
			root, err := ssz.SigningRoot(b)
			if err != nil {
				return nil, err
			}
			containers[i] = &ethpb.BeaconBlockContainer{
				Block:     b,
				BlockRoot: root[:],
			}
		}

		return &ethpb.ListBlocksResponse{
			BlockContainers: containers,
			TotalSize:       int32(numBlks),
			NextPageToken:   nextPageToken,
		}, nil

	case *ethpb.ListBlocksRequest_Root:
		blk, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(q.Root))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve block: %v", err)
		}
		if blk == nil {
			return &ethpb.ListBlocksResponse{
				BlockContainers: make([]*ethpb.BeaconBlockContainer, 0),
				TotalSize:       0,
				NextPageToken:   strconv.Itoa(0),
			}, nil
		}
		root, err := ssz.SigningRoot(blk)
		if err != nil {
			return nil, err
		}

		return &ethpb.ListBlocksResponse{
			BlockContainers: []*ethpb.BeaconBlockContainer{{
				Block:     blk,
				BlockRoot: root[:]},
			},
			TotalSize: 1,
		}, nil

	case *ethpb.ListBlocksRequest_Slot:
		blks, err := bs.BeaconDB.Blocks(ctx, filters.NewFilter().SetStartSlot(q.Slot).SetEndSlot(q.Slot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for slot %d: %v", q.Slot, err)
		}

		numBlks := len(blks)
		if numBlks == 0 {
			return &ethpb.ListBlocksResponse{
				BlockContainers: make([]*ethpb.BeaconBlockContainer, 0),
				TotalSize:       0,
				NextPageToken:   strconv.Itoa(0),
			}, nil
		}

		start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not paginate blocks: %v", err)
		}

		returnedBlks := blks[start:end]
		containers := make([]*ethpb.BeaconBlockContainer, len(returnedBlks))
		for i, b := range returnedBlks {
			root, err := ssz.SigningRoot(b)
			if err != nil {
				return nil, err
			}
			containers[i] = &ethpb.BeaconBlockContainer{
				Block:     b,
				BlockRoot: root[:],
			}
		}

		return &ethpb.ListBlocksResponse{
			BlockContainers: containers,
			TotalSize:       int32(numBlks),
			NextPageToken:   nextPageToken,
		}, nil
	}

	return nil, status.Error(codes.InvalidArgument, "Must specify a filter criteria for fetching blocks")
}

// GetChainHead retrieves information about the head of the beacon chain from
// the view of the beacon chain node.
//
// This includes the head block slot and root as well as information about
// the most recent finalized and justified slots.
func (bs *Server) GetChainHead(ctx context.Context, _ *ptypes.Empty) (*ethpb.ChainHead, error) {
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	headBlock := bs.HeadFetcher.HeadBlock()
	headBlockRoot, err := ssz.SigningRoot(headBlock)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head block root: %v", err)
	}

	finalizedCheckpoint := headState.FinalizedCheckpoint
	b, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(finalizedCheckpoint.Root))
	if err != nil || b == nil {
		return nil, status.Error(codes.Internal, "Could not get finalized block")
	}
	finalizedBlockSlot := b.Slot

	justifiedCheckpoint := headState.CurrentJustifiedCheckpoint
	b, err = bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(justifiedCheckpoint.Root))
	if err != nil || b == nil {
		return nil, status.Error(codes.Internal, "Could not get justified block")
	}
	justifiedBlockSlot := b.Slot

	prevJustifiedCheckpoint := headState.PreviousJustifiedCheckpoint
	b, err = bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(prevJustifiedCheckpoint.Root))
	if err != nil || b == nil {
		return nil, status.Error(codes.Internal, "Could not get prev justified block")
	}
	prevJustifiedBlockSlot := b.Slot

	return &ethpb.ChainHead{
		HeadBlockSlot:              headBlock.Slot,
		HeadBlockEpoch:             helpers.SlotToEpoch(headBlock.Slot),
		HeadBlockRoot:              headBlockRoot[:],
		FinalizedBlockRoot:         finalizedCheckpoint.Root,
		FinalizedBlockSlot:         finalizedBlockSlot,
		FinalizedEpoch:             finalizedCheckpoint.Epoch,
		JustifiedBlockRoot:         justifiedCheckpoint.Root,
		JustifiedBlockSlot:         justifiedBlockSlot,
		JustifiedEpoch:             justifiedCheckpoint.Epoch,
		PreviousJustifiedBlockRoot: prevJustifiedCheckpoint.Root,
		PreviousJustifiedSlot:      prevJustifiedBlockSlot,
		PreviousJustifiedEpoch:     prevJustifiedCheckpoint.Epoch,
	}, nil
}
