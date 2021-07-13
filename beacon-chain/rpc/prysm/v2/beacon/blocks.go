package beacon

import (
	"context"
	"strconv"

	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/interfaces/version"
	"github.com/prysmaticlabs/prysm/shared/pagination"
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
) (*prysmv2.ListBlocksResponseAltair, error) {
	if int(req.PageSize) > cmd.Get().MaxRPCPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, cmd.Get().MaxRPCPageSize)
	}

	switch q := req.QueryFilter.(type) {
	case *ethpb.ListBlocksRequest_Epoch:
		blks, _, err := bs.BeaconDB.Blocks(ctx, filters.NewFilter().SetStartEpoch(q.Epoch).SetEndEpoch(q.Epoch))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get blocks: %v", err)
		}

		numBlks := len(blks)
		if numBlks == 0 {
			return &prysmv2.ListBlocksResponseAltair{
				BlockContainers: make([]*prysmv2.BeaconBlockContainerAltair, 0),
				TotalSize:       0,
				NextPageToken:   strconv.Itoa(0),
			}, nil
		}

		start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not paginate blocks: %v", err)
		}

		returnedBlks := blks[start:end]
		containers := make([]*prysmv2.BeaconBlockContainerAltair, len(returnedBlks))
		for i, b := range returnedBlks {
			root, err := b.Block().HashTreeRoot()
			if err != nil {
				return nil, err
			}
			canonical, err := bs.CanonicalFetcher.IsCanonical(ctx, root)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not determine if block is canonical: %v", err)
			}
			ctr, err := convertToBlockContainer(b, root, canonical)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get block container: %v", err)
			}
			containers[i] = ctr
		}

		return &prysmv2.ListBlocksResponseAltair{
			BlockContainers: containers,
			TotalSize:       int32(numBlks),
			NextPageToken:   nextPageToken,
		}, nil
	case *ethpb.ListBlocksRequest_Root:
		blk, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(q.Root))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve block: %v", err)
		}
		if blk == nil || blk.IsNil() {
			return &prysmv2.ListBlocksResponseAltair{
				BlockContainers: make([]*prysmv2.BeaconBlockContainerAltair, 0),
				TotalSize:       0,
				NextPageToken:   strconv.Itoa(0),
			}, nil
		}
		root, err := blk.Block().HashTreeRoot()
		if err != nil {
			return nil, err
		}
		canonical, err := bs.CanonicalFetcher.IsCanonical(ctx, root)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not determine if block is canonical: %v", err)
		}
		ctr, err := convertToBlockContainer(blk, root, canonical)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get block container: %v", err)
		}
		return &prysmv2.ListBlocksResponseAltair{
			BlockContainers: []*prysmv2.BeaconBlockContainerAltair{ctr},
			TotalSize:       1,
		}, nil

	case *ethpb.ListBlocksRequest_Slot:
		hasBlocks, blks, err := bs.BeaconDB.BlocksBySlot(ctx, q.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for slot %d: %v", q.Slot, err)
		}
		if !hasBlocks {
			return &prysmv2.ListBlocksResponseAltair{
				BlockContainers: make([]*prysmv2.BeaconBlockContainerAltair, 0),
				TotalSize:       0,
				NextPageToken:   strconv.Itoa(0),
			}, nil
		}

		numBlks := len(blks)

		start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not paginate blocks: %v", err)
		}

		returnedBlks := blks[start:end]
		containers := make([]*prysmv2.BeaconBlockContainerAltair, len(returnedBlks))
		for i, b := range returnedBlks {
			root, err := b.Block().HashTreeRoot()
			if err != nil {
				return nil, err
			}
			canonical, err := bs.CanonicalFetcher.IsCanonical(ctx, root)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not determine if block is canonical: %v", err)
			}
			ctr, err := convertToBlockContainer(b, root, canonical)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get block container: %v", err)
			}
			containers[i] = ctr
		}

		return &prysmv2.ListBlocksResponseAltair{
			BlockContainers: containers,
			TotalSize:       int32(numBlks),
			NextPageToken:   nextPageToken,
		}, nil
	case *ethpb.ListBlocksRequest_Genesis:
		genBlk, err := bs.BeaconDB.GenesisBlock(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for genesis slot: %v", err)
		}
		if genBlk == nil || genBlk.IsNil() {
			return nil, status.Error(codes.Internal, "Could not find genesis block")
		}
		root, err := genBlk.Block().HashTreeRoot()
		if err != nil {
			return nil, err
		}
		ctr, err := convertToBlockContainer(genBlk, root, true)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get block container: %v", err)
		}
		containers := []*prysmv2.BeaconBlockContainerAltair{ctr}

		return &prysmv2.ListBlocksResponseAltair{
			BlockContainers: containers,
			TotalSize:       int32(1),
			NextPageToken:   strconv.Itoa(0),
		}, nil
	}

	return nil, status.Error(codes.InvalidArgument, "Must specify a filter criteria for fetching blocks")
}

func convertToBlockContainer(blk interfaces.SignedBeaconBlock, root [32]byte, isCanonical bool) (*prysmv2.BeaconBlockContainerAltair, error) {
	ctr := &prysmv2.BeaconBlockContainerAltair{
		BlockRoot: root[:],
		Canonical: isCanonical,
	}

	switch blk.Version() {
	case version.Phase0:
		rBlk, err := blk.PbPhase0Block()
		if err != nil {
			return nil, err
		}
		ctr.Block = &prysmv2.BeaconBlockContainerAltair_Phase0Block{Phase0Block: rBlk}
	case version.Altair:
		rBlk, err := blk.PbAltairBlock()
		if err != nil {
			return nil, err
		}
		ctr.Block = &prysmv2.BeaconBlockContainerAltair_AltairBlock{AltairBlock: rBlk}
	}
	return ctr, nil
}
