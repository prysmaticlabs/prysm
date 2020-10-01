package beacon_v1

import (
	"context"
	"encoding/hex"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (bs *Server) GetBlockHeader(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockHeaderResponse, error) {
	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if blk == nil {
		return nil, status.Errorf(codes.Internal, "Could not find requested block")
	}

	blkHdr, err := blockutil.SignedBeaconBlockHeaderFromBlock(blk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block header from block: %v", err)
	}
	marshaledBlkHdr, err := blkHdr.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal block: %v", err)
	}
	v1BlockHdr := &ethpb.SignedBeaconBlockHeader{}
	if err := proto.Unmarshal(marshaledBlkHdr, v1BlockHdr); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal block: %v", err)
	}
	root, err := v1BlockHdr.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal block header: %v", err)
	}

	return &ethpb.BlockHeaderResponse{
		Data: &ethpb.BlockHeaderContainer{
			Root:      root[:],
			Canonical: true,
			Header: &ethpb.BeaconBlockHeaderContainer{
				Message:   v1BlockHdr.Header,
				Signature: v1BlockHdr.Signature,
			},
		},
	}, nil
}

func (bs *Server) ListBlockHeaders(ctx context.Context, req *ethpb.BlockHeadersRequest) (*ethpb.BlockHeadersResponse, error) {
	return &ethpb.BlockHeadersResponse{}, nil
}

func (bs *Server) SubmitBlock(ctx context.Context, req *ethpb.BeaconBlockContainer) (*ptypes.Empty, error) {
	blk := req.Message
	root, err := blk.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not tree hash block: %v", err)
	}

	// Do not block proposal critical path with debug logging or block feed updates.
	defer func() {
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(root[:]))).Debugf(
			"Block proposal received via RPC")
		bs.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: &blockfeed.ReceivedBlockData{SignedBlock: blk},
		})
	}()

	// Broadcast the new block to the network.
	if err := bs.Broadcaster.Broadcast(ctx, blk); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast block: %v", err)
	}
	log.WithFields(logrus.Fields{
		"blockRoot": hex.EncodeToString(root[:]),
	}).Debug("Broadcasting block")

	if err := bs.BlockReceiver.ReceiveBlock(ctx, blk, root); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process beacon block: %v", err)
	}

	return &ptypes.Empty{}, nil
}

func (bs *Server) GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockResponse, error) {
	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if blk == nil {
		return nil, status.Errorf(codes.Internal, "Could not find requested block")
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
					Root: req.BlockId,
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
}

func (bs *Server) ListBlockAttestations(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockAttestationsResponse, error) {
	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if blk == nil {
		return nil, status.Errorf(codes.Internal, "Could not find requested block")
	}

	marshaledBlk, err := blk.Block.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal block: %v", err)
	}
	v1Block := &ethpb.BeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1Block); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal block: %v", err)
	}
	return &ethpb.BlockAttestationsResponse{
		Data: v1Block.Body.Attestations,
	}, nil
}

func (bs *Server) blockFromBlockID(ctx context.Context, blockId []byte) (*ethpb_alpha.SignedBeaconBlock, error) {
	var err error
	var blk *ethpb_alpha.SignedBeaconBlock
	switch string(blockId) {
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
		if len(blockId) == 32 {
			blk, err = bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(blockId))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve block: %v", err)
			}
			if blk == nil {
				return nil, nil
			}
		} else {
			slot := bytesutil.FromBytes8(blockId)
			fmt.Println(slot)
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
	return blk, nil
}
