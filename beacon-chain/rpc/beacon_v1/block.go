package beacon_v1

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (bs *Server) GetBlockHeader(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockHeaderResponse, error) {
	return &ethpb.BlockHeaderResponse{}, nil
}

func (bs *Server) ListBlockHeaders(ctx context.Context, req *ethpb.BlockHeadersRequest) (*ethpb.BlockHeadersResponse, error) {
	return &ethpb.BlockHeadersResponse{}, nil
}

func (bs *Server) SubmitBlock(ctx context.Context, req *ethpb.BeaconBlockContainer) (*ptypes.Empty, error) {
	return &ptypes.Empty{}, nil
}

func (bs *Server) GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockResponse, error) {
	var err error
	var blk *ethpb_alpha.SignedBeaconBlock
	switch string(req.BlockId) {
	case "head":
		blk, err = bs.HeadFetcher.HeadBlock(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve head block: %v", err)
		}
		if blk == nil {
			return nil, status.Errorf(codes.Internal, "No head block was found")
		}
	case "finalized":
		finalized, err := bs.BeaconDB.FinalizedCheckpoint(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve finalized checkpoint: %v", err)
		}
		finalizedRoot := bytesutil.ToBytes32(finalized.Root)
		blk, err = bs.BeaconDB.Block(ctx, finalizedRoot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get finalized block from db")
		}
		if blk == nil {
			return nil, status.Errorf(codes.Internal, "Could not find finalized block")
		}
	case "genesis":
		blk, err = bs.BeaconDB.GenesisBlock(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for genesis slot: %v", err)
		}
		if blk == nil {
			return nil, status.Error(codes.Internal, "Could not find genesis block")
		}
	default:
		if len(req.BlockId) == 32 {
			blk, err = bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(req.BlockId))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve block: %v", err)
			}
			if blk == nil {
				return nil, nil
			}
		} else {
			slot := bytesutil.BytesToUint64BigEndian(req.BlockId)
			blks, err := bs.BeaconDB.Blocks(ctx, filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for slot %d: %v", slot, err)
			}

			numBlks := len(blks)
			if numBlks == 0 {
				return nil, nil
			}
			blk = blks[0]
		}
	}

	marshaledBlk, err := blk.Block.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal block: %v", err)
	}
	v1Block := &ethpb.BeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1Block); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal block: %v", err)
	}

	return &ethpb.BlockResponse{
		Data: &ethpb.BeaconBlockContainer{
			Message:   v1Block,
			Signature: blk.Signature,
		},
	}, nil
}

func (bs *Server) GetBlockRoot(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockRootResponse, error) {
	switch string(req.BlockId) {
	case "head":
		root, err := bs.HeadFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve head block: %v", err)
		}
		if root == nil {
			return nil, status.Errorf(codes.Internal, "No head root was found")
		}
		return &ethpb.BlockRootResponse{
			Data: &ethpb.BlockRootContainer{
				Root: root,
			},
		}, nil
	case "finalized":
		finalized := bs.FinalizationFetcher.FinalizedCheckpt()
		if finalized == nil {
			return nil, status.Errorf(codes.Internal, "No finalized root was found")
		}
		return &ethpb.BlockRootResponse{
			Data: &ethpb.BlockRootContainer{
				Root: finalized.Root,
			},
		}, nil
	case "genesis":
		blk, err := bs.BeaconDB.GenesisBlock(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for genesis slot: %v", err)
		}
		if blk == nil {
			return nil, status.Error(codes.Internal, "Could not find genesis block")
		}
		root, err := blk.HashTreeRoot()
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not hash genesis block")
		}
		return &ethpb.BlockRootResponse{
			Data: &ethpb.BlockRootContainer{
				Root: root[:],
			},
		}, nil
	default:
		if len(req.BlockId) == 32 {
			return &ethpb.BlockRootResponse{
				Data: &ethpb.BlockRootContainer{
					Root: req.BlockId[:],
				},
			}, nil
		} else {
			slot := bytesutil.BytesToUint64BigEndian(req.BlockId)
			roots, err := bs.BeaconDB.BlockRoots(ctx, filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for slot %d: %v", slot, err)
			}

			numRoots := len(roots)
			if numRoots == 0 {
				return nil, nil
			}
			root := roots[0]
			return &ethpb.BlockRootResponse{
				Data: &ethpb.BlockRootContainer{
					Root: root[:],
				},
			}, nil
		}
	}

	marshaledBlk, err := blk.Block.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal block: %v", err)
	}
	v1Block := &ethpb.BeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1Block); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal block: %v", err)
	}

	return &ethpb.BlockResponse{
		Data: &ethpb.BeaconBlockContainer{
			Message:   v1Block,
			Signature: blk.Signature,
		},
	}, nil
	return &ethpb.BlockRootResponse{}, nil
}

func (bs *Server) ListBlockAttestations(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockAttestationsResponse, error) {
	return &ethpb.BlockAttestationsResponse{}, nil
}
