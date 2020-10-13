package beaconv1

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBlockHeader retrieves block header for given block id.
func (bs *Server) GetBlockHeader(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockHeaderResponse, error) {
	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if blk == nil {
		return nil, status.Errorf(codes.NotFound, "Could not find requested block")
	}

	blkHdr, err := blockutil.SignedBeaconBlockHeaderFromBlock(blk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block header from block: %v", err)
	}
	marshaledBlkHdr, err := blkHdr.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal block header: %v", err)
	}
	v1BlockHdr := &ethpb.SignedBeaconBlockHeader{}
	if err := proto.Unmarshal(marshaledBlkHdr, v1BlockHdr); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal block header: %v", err)
	}
	root, err := v1BlockHdr.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not hash block header: %v", err)
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

// ListBlockHeaders retrieves block headers matching given query. By default it will fetch current head slot blocks.
func (bs *Server) ListBlockHeaders(ctx context.Context, req *ethpb.BlockHeadersRequest) (*ethpb.BlockHeadersResponse, error) {
	var err error
	var blks []*ethpb_alpha.SignedBeaconBlock
	if len(req.ParentRoot) == 32 {
		blks, err = bs.BeaconDB.Blocks(ctx, filters.NewFilter().SetParentRoot(req.ParentRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks: %v", err)
		}
	} else {
		blks, err = bs.BeaconDB.Blocks(ctx, filters.NewFilter().SetStartSlot(req.Slot).SetEndSlot(req.Slot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for slot %d: %v", req.Slot, err)
		}
	}
	if blks == nil {
		return nil, status.Error(codes.NotFound, "Could not find requested blocks")
	}

	blkHdrs := make([]*ethpb.BlockHeaderContainer, len(blks))
	for i, blk := range blks {
		blkHdr, err := v1alpha1BlockToV1BlockHeader(blk)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get block header from block: %v", err)
		}
		root, err := blkHdr.Header.HashTreeRoot()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not hash block header: %v", err)
		}
		blkHdrs[i] = &ethpb.BlockHeaderContainer{
			Root: root[:],
			Header: &ethpb.BeaconBlockHeaderContainer{
				Message:   blkHdr.Header,
				Signature: blkHdr.Signature,
			},
		}
	}

	return &ethpb.BlockHeadersResponse{Data: blkHdrs}, nil
}

// SubmitBlock instructs the beacon node to broadcast a newly signed beacon block to the beacon network, to be
// included in the beacon chain. The beacon node is not required to validate the signed BeaconBlock, and a successful
// response (20X) only indicates that the broadcast has been successful. The beacon node is expected to integrate the
// new block into its state, and therefore validate the block internally, however blocks which fail the validation are
// still broadcast but a different status code is returned (202).
func (bs *Server) SubmitBlock(ctx context.Context, req *ethpb.BeaconBlockContainer) (*ptypes.Empty, error) {
	blk := req.Message
	root, err := blk.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not tree hash block: %v", err)
	}

	v1alpha1Block, err := v1ToV1alpha1Block(&ethpb.SignedBeaconBlock{Block: blk, Signature: req.Signature})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert block to v1")
	}

	// Do not block proposal critical path with debug logging or block feed updates.
	defer func() {
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(root[:]))).Debugf(
			"Block proposal received via RPC")
		bs.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: &blockfeed.ReceivedBlockData{SignedBlock: v1alpha1Block},
		})
	}()

	// Broadcast the new block to the network.
	if err := bs.Broadcaster.Broadcast(ctx, v1alpha1Block); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast block: %v", err)
	}
	log.WithFields(logrus.Fields{
		"blockRoot": hex.EncodeToString(root[:]),
	}).Debug("Broadcasting block")

	if err := bs.BlockReceiver.ReceiveBlock(ctx, v1alpha1Block, root); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process beacon block: %v", err)
	}

	return &ptypes.Empty{}, nil
}

// GetBlock retrieves block details for given block id.
func (bs *Server) GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockResponse, error) {
	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if blk == nil {
		return nil, status.Errorf(codes.NotFound, "Could not find requested block")
	}

	v1Block, err := v1alpha1ToV1Block(blk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert block to v1")
	}

	return &ethpb.BlockResponse{
		Data: &ethpb.BeaconBlockContainer{
			Message:   v1Block.Block,
			Signature: blk.Signature,
		},
	}, nil
}

// GetBlockRoot retrieves hashTreeRoot of BeaconBlock/BeaconBlockHeader.
func (bs *Server) GetBlockRoot(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockRootResponse, error) {
	var root []byte
	var err error
	switch string(req.BlockId) {
	case "head":
		root, err = bs.HeadFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve head block: %v", err)
		}
		if root == nil {
			return nil, status.Errorf(codes.NotFound, "No head root was found")
		}
	case "finalized":
		finalized := bs.FinalizationFetcher.FinalizedCheckpt()
		if finalized == nil {
			return nil, status.Errorf(codes.NotFound, "No finalized root was found")
		}
		root = finalized.Root
	case "genesis":
		blk, err := bs.BeaconDB.GenesisBlock(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for genesis slot: %v", err)
		}
		if blk == nil {
			return nil, status.Error(codes.NotFound, "Could not find genesis block")
		}
		blkRoot, err := blk.Block.HashTreeRoot()
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not hash genesis block")
		}
		root = blkRoot[:]
	default:
		if len(req.BlockId) == 32 {
			root = req.BlockId
		} else {
			slot := bytesutil.BytesToUint64BigEndian(req.BlockId)
			roots, err := bs.BeaconDB.BlockRoots(ctx, filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for slot %d: %v", slot, err)
			}

			numRoots := len(roots)
			if numRoots == 0 {
				return nil, status.Error(codes.NotFound, "Could not find any blocks with given root")
			}
			root = roots[0][:]
		}
	}

	return &ethpb.BlockRootResponse{
		Data: &ethpb.BlockRootContainer{
			Root: root,
		},
	}, nil
}

// ListBlockAttestations retrieves attestation included in requested block.
func (bs *Server) ListBlockAttestations(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockAttestationsResponse, error) {
	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if blk == nil {
		return nil, status.Errorf(codes.NotFound, "Could not find requested block")
	}

	v1Block, err := v1alpha1ToV1Block(blk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert block to v1")
	}
	return &ethpb.BlockAttestationsResponse{
		Data: v1Block.Block.Body.Attestations,
	}, nil
}

func (bs *Server) blockFromBlockID(ctx context.Context, blockId []byte) (*ethpb_alpha.SignedBeaconBlock, error) {
	var err error
	var blk *ethpb_alpha.SignedBeaconBlock
	switch string(blockId) {
	case "head":
		blk, err = bs.HeadFetcher.HeadBlock(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve head block")
		}
	case "finalized":
		finalized := bs.FinalizationFetcher.FinalizedCheckpt()
		finalizedRoot := bytesutil.ToBytes32(finalized.Root)
		blk, err = bs.BeaconDB.Block(ctx, finalizedRoot)
		if err != nil {
			return nil, errors.New("could not get finalized block from db")
		}
	case "genesis":
		blk, err = bs.BeaconDB.GenesisBlock(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve blocks for genesis slot")
		}
	default:
		if len(blockId) == 32 {
			blk, err = bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(blockId))
			if err != nil {
				return nil, errors.Wrap(err, "could not retrieve block")
			}
		} else {
			slotInt, err := strconv.Atoi(string(blockId))
			if err != nil {
				return nil, errors.Wrap(err, "could not decode block id")
			}
			slot := uint64(slotInt)
			blks, err := bs.BeaconDB.Blocks(ctx, filters.NewFilter().SetStartSlot(slot).SetEndSlot(slot))
			if err != nil {
				return nil, errors.Wrapf(err, "could not retrieve blocks for slot %d", slot)
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

func v1alpha1BlockToV1BlockHeader(block *ethpb_alpha.SignedBeaconBlock) (*ethpb.SignedBeaconBlockHeader, error) {
	bodyRoot, err := stateutil.BlockBodyRoot(block.Block.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get body root of block")
	}
	return &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          block.Block.Slot,
			ProposerIndex: block.Block.ProposerIndex,
			ParentRoot:    block.Block.ParentRoot,
			StateRoot:     block.Block.StateRoot,
			BodyRoot:      bodyRoot[:],
		},
		Signature: block.Signature,
	}, nil
}

func v1alpha1ToV1Block(alphaBlk *ethpb_alpha.SignedBeaconBlock) (*ethpb.SignedBeaconBlock, error) {
	marshaledBlk, err := alphaBlk.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1Block := &ethpb.SignedBeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1Block, nil
}

func v1ToV1alpha1Block(alphaBlk *ethpb.SignedBeaconBlock) (*ethpb_alpha.SignedBeaconBlock, error) {
	marshaledBlk, err := alphaBlk.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpb_alpha.SignedBeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}
