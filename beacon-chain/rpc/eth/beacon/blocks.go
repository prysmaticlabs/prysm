package beacon

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// blockIdParseError represents an error scenario where a block ID could not be parsed.
type blockIdParseError struct {
	message string
}

// newBlockIdParseError creates a new error instance.
func newBlockIdParseError(reason error) blockIdParseError {
	return blockIdParseError{
		message: errors.Wrapf(reason, "could not parse block ID").Error(),
	}
}

// Error returns the underlying error message.
func (e *blockIdParseError) Error() string {
	return e.message
}

// GetBlockHeader retrieves block header for given block id.
func (bs *Server) GetBlockHeader(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockHeaderResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetBlockHeader")
	defer span.End()

	rBlk, err := bs.blockFromBlockID(ctx, req.BlockId)
	if invalidBlockIdErr, ok := err.(*blockIdParseError); ok {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid block ID: %v", invalidBlockIdErr)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if rBlk == nil || rBlk.IsNil() {
		return nil, status.Errorf(codes.NotFound, "Could not find requested block header")
	}
	blk, err := rBlk.PbPhase0Block()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get raw block: %v", err)
	}

	v1BlockHdr, err := migration.V1Alpha1BlockToV1BlockHeader(blk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block header from block: %v", err)
	}

	blkRoot, err := blk.Block.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not hash block: %v", err)
	}
	canonical, err := bs.ChainInfoFetcher.IsCanonical(ctx, blkRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if block root is canonical: %v", err)
	}
	root, err := v1BlockHdr.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not hash block header: %v", err)
	}

	return &ethpbv1.BlockHeaderResponse{
		Data: &ethpbv1.BlockHeaderContainer{
			Root:      root[:],
			Canonical: canonical,
			Header: &ethpbv1.BeaconBlockHeaderContainer{
				Message:   v1BlockHdr.Message,
				Signature: v1BlockHdr.Signature,
			},
		},
	}, nil
}

// ListBlockHeaders retrieves block headers matching given query. By default it will fetch current head slot blocks.
func (bs *Server) ListBlockHeaders(ctx context.Context, req *ethpbv1.BlockHeadersRequest) (*ethpbv1.BlockHeadersResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.ListBlockHeaders")
	defer span.End()

	var err error
	var blks []block.SignedBeaconBlock
	var blkRoots [][32]byte
	if len(req.ParentRoot) == 32 {
		blks, blkRoots, err = bs.BeaconDB.Blocks(ctx, filters.NewFilter().SetParentRoot(req.ParentRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks: %v", err)
		}
	} else {
		slot := bs.ChainInfoFetcher.HeadSlot()
		if req.Slot != nil {
			slot = *req.Slot
		}
		_, blks, err = bs.BeaconDB.BlocksBySlot(ctx, slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for slot %d: %v", req.Slot, err)
		}
		_, blkRoots, err = bs.BeaconDB.BlockRootsBySlot(ctx, slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve block roots for slot %d: %v", req.Slot, err)
		}
	}
	if len(blks) == 0 {
		return nil, status.Error(codes.NotFound, "Could not find requested blocks")
	}

	blkHdrs := make([]*ethpbv1.BlockHeaderContainer, len(blks))
	for i, bl := range blks {
		blk, err := bl.PbPhase0Block()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get raw block: %v", err)
		}
		blkHdr, err := migration.V1Alpha1BlockToV1BlockHeader(blk)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get block header from block: %v", err)
		}
		canonical, err := bs.ChainInfoFetcher.IsCanonical(ctx, blkRoots[i])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not determine if block root is canonical: %v", err)
		}
		root, err := blkHdr.Message.HashTreeRoot()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not hash block header: %v", err)
		}
		blkHdrs[i] = &ethpbv1.BlockHeaderContainer{
			Root:      root[:],
			Canonical: canonical,
			Header: &ethpbv1.BeaconBlockHeaderContainer{
				Message:   blkHdr.Message,
				Signature: blkHdr.Signature,
			},
		}
	}

	return &ethpbv1.BlockHeadersResponse{Data: blkHdrs}, nil
}

// SubmitBlock instructs the beacon node to broadcast a newly signed beacon block to the beacon network, to be
// included in the beacon chain. The beacon node is not required to validate the signed BeaconBlock, and a successful
// response (20X) only indicates that the broadcast has been successful. The beacon node is expected to integrate the
// new block into its state, and therefore validate the block internally, however blocks which fail the validation are
// still broadcast but a different status code is returned (202).
func (bs *Server) SubmitBlock(ctx context.Context, req *ethpbv1.BeaconBlockContainer) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.SubmitBlock")
	defer span.End()

	blk := req.Message
	rBlock, err := migration.V1ToV1Alpha1SignedBlock(&ethpbv1.SignedBeaconBlock{Block: blk, Signature: req.Signature})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not convert block to v1 block")
	}
	v1alpha1Block := wrapper.WrappedPhase0SignedBeaconBlock(rBlock)

	root, err := blk.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not tree hash block: %v", err)
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
	if err := bs.Broadcaster.Broadcast(ctx, v1alpha1Block.Proto()); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast block: %v", err)
	}

	if err := bs.BlockReceiver.ReceiveBlock(ctx, v1alpha1Block, root); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process beacon block: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetBlock retrieves block details for given block ID.
func (bs *Server) GetBlock(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetBlock")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlock(blk, err)
	if err != nil {
		return nil, err
	}
	signedBeaconBlock, err := migration.SignedBeaconBlock(blk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}

	return &ethpbv1.BlockResponse{
		Data: &ethpbv1.BeaconBlockContainer{
			Message:   signedBeaconBlock.Block,
			Signature: signedBeaconBlock.Signature,
		},
	}, nil
}

// GetBlockSSZ returns the SSZ-serialized version of the becaon block for given block ID.
func (bs *Server) GetBlockSSZ(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockSSZResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetBlockSSZ")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlock(blk, err)
	if err != nil {
		return nil, err
	}
	signedBeaconBlock, err := migration.SignedBeaconBlock(blk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	sszBlock, err := signedBeaconBlock.MarshalSSZ()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ: %v", err)
	}

	return &ethpbv1.BlockSSZResponse{Data: sszBlock}, nil
}

// GetBlockV2 retrieves block details for given block ID.
func (bs *Server) GetBlockV2(ctx context.Context, req *ethpbv2.BlockRequestV2) (*ethpbv2.BlockResponseV2, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockAltair")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlock(blk, err)
	if err != nil {
		return nil, err
	}
	phase0Blk, err := blk.PbPhase0Block()
	// Assume we have an Altair block when Phase 0 block is unsupported.
	// In such case we continue with the rest of the function.
	if err != nil && !errors.Is(err, wrapper.ErrUnsupportedPhase0Block) {
		return nil, status.Errorf(codes.Internal, "Could not check for phase 0 block")
	}

	if phase0Blk != nil {
		v1Blk, err := migration.SignedBeaconBlock(blk)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		return &ethpbv2.BlockResponseV2{
			Data: &ethpbv2.BeaconBlockContainerV2{
				Block:     &ethpbv2.BeaconBlockContainerV2_Phase0Block{Phase0Block: v1Blk.Block},
				Signature: v1Blk.Signature,
			},
		}, nil
	}
	altairBlk, err := blk.PbAltairBlock()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check for Altair block")
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlk.Block)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	return &ethpbv2.BlockResponseV2{
		Data: &ethpbv2.BeaconBlockContainerV2{
			Block:     &ethpbv2.BeaconBlockContainerV2_AltairBlock{AltairBlock: v2Blk},
			Signature: blk.Signature(),
		},
	}, nil
}

// GetBlockSSZV2 returns the SSZ-serialized version of the beacon block for given block ID.
func (bs *Server) GetBlockSSZV2(ctx context.Context, req *ethpbv2.BlockRequestV2) (*ethpbv2.BlockSSZResponseV2, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockSSZV2")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlock(blk, err)
	if err != nil {
		return nil, err
	}
	phase0Blk, err := blk.PbPhase0Block()
	// Assume we have an Altair block when Phase 0 block is unsupported.
	// In such case we continue with the rest of the function.
	if err != nil && !errors.Is(err, wrapper.ErrUnsupportedPhase0Block) {
		return nil, status.Errorf(codes.Internal, "Could not check for phase 0 block")
	}

	if phase0Blk != nil {
		signedBeaconBlock, err := migration.SignedBeaconBlock(blk)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		sszBlock, err := signedBeaconBlock.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ: %v", err)
		}
		return &ethpbv2.BlockSSZResponseV2{Data: sszBlock}, nil
	}
	altairBlk, err := blk.PbAltairBlock()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check for Altair block")
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlk.Block)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	data := &ethpbv2.SignedBeaconBlockAltair{
		Message:   v2Blk,
		Signature: blk.Signature(),
	}
	sszData, err := data.MarshalSSZ()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ: %v", err)
	}
	return &ethpbv2.BlockSSZResponseV2{Data: sszData}, nil
}

// GetBlockRoot retrieves hashTreeRoot of BeaconBlock/BeaconBlockHeader.
func (bs *Server) GetBlockRoot(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockRootResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetBlockRoot")
	defer span.End()

	var root []byte
	var err error
	switch string(req.BlockId) {
	case "head":
		root, err = bs.ChainInfoFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve head block: %v", err)
		}
		if root == nil {
			return nil, status.Errorf(codes.NotFound, "No head root was found")
		}
	case "finalized":
		finalized := bs.ChainInfoFetcher.FinalizedCheckpt()
		root = finalized.Root
	case "genesis":
		blk, err := bs.BeaconDB.GenesisBlock(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for genesis slot: %v", err)
		}
		if blk == nil || blk.IsNil() {
			return nil, status.Error(codes.NotFound, "Could not find genesis block")
		}
		blkRoot, err := blk.Block().HashTreeRoot()
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not hash genesis block")
		}
		root = blkRoot[:]
	default:
		if len(req.BlockId) == 32 {
			block, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(req.BlockId))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve block for block root %#x: %v", req.BlockId, err)
			}
			if block == nil || block.IsNil() {
				return nil, status.Error(codes.NotFound, "Could not find any blocks with given root")
			}

			root = req.BlockId
		} else {
			slot, err := strconv.ParseUint(string(req.BlockId), 10, 64)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "Could not parse block ID: %v", err)
			}
			hasRoots, roots, err := bs.BeaconDB.BlockRootsBySlot(ctx, types.Slot(slot))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve blocks for slot %d: %v", slot, err)
			}

			if !hasRoots {
				return nil, status.Error(codes.NotFound, "Could not find any blocks with given slot")
			}
			root = roots[0][:]
			if len(roots) == 1 {
				break
			}
			for _, blockRoot := range roots {
				canonical, err := bs.ChainInfoFetcher.IsCanonical(ctx, blockRoot)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "Could not determine if block root is canonical: %v", err)
				}
				if canonical {
					root = blockRoot[:]
					break
				}
			}
		}
	}

	return &ethpbv1.BlockRootResponse{
		Data: &ethpbv1.BlockRootContainer{
			Root: root,
		},
	}, nil
}

// ListBlockAttestations retrieves attestation included in requested block.
func (bs *Server) ListBlockAttestations(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockAttestationsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.ListBlockAttestations")
	defer span.End()

	rBlk, err := bs.blockFromBlockID(ctx, req.BlockId)
	if invalidBlockIdErr, ok := err.(*blockIdParseError); ok {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid block ID: %v", invalidBlockIdErr)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if rBlk == nil || rBlk.IsNil() {
		return nil, status.Errorf(codes.NotFound, "Could not find requested block")
	}

	blk, err := rBlk.PbPhase0Block()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get raw block: %v", err)
	}

	v1Block, err := migration.V1Alpha1ToV1SignedBlock(blk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert block to v1 block")
	}
	return &ethpbv1.BlockAttestationsResponse{
		Data: v1Block.Block.Body.Attestations,
	}, nil
}

func (bs *Server) blockFromBlockID(ctx context.Context, blockId []byte) (block.SignedBeaconBlock, error) {
	var err error
	var blk block.SignedBeaconBlock
	switch string(blockId) {
	case "head":
		blk, err = bs.ChainInfoFetcher.HeadBlock(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve head block")
		}
	case "finalized":
		finalized := bs.ChainInfoFetcher.FinalizedCheckpt()
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
			slot, err := strconv.ParseUint(string(blockId), 10, 64)
			if err != nil {
				e := newBlockIdParseError(err)
				return nil, &e
			}
			_, blks, err := bs.BeaconDB.BlocksBySlot(ctx, types.Slot(slot))
			if err != nil {
				return nil, errors.Wrapf(err, "could not retrieve blocks for slot %d", slot)
			}
			_, roots, err := bs.BeaconDB.BlockRootsBySlot(ctx, types.Slot(slot))
			if err != nil {
				return nil, errors.Wrapf(err, "could not retrieve block roots for slot %d", slot)
			}

			numBlks := len(blks)
			if numBlks == 0 {
				return nil, nil
			}
			blk = blks[0]
			if numBlks == 1 {
				break
			}
			for i, block := range blks {
				canonical, err := bs.ChainInfoFetcher.IsCanonical(ctx, roots[i])
				if err != nil {
					return nil, status.Errorf(codes.Internal, "Could not determine if block root is canonical: %v", err)
				}
				if canonical {
					blk = block
					break
				}
			}
		}
	}
	return blk, nil
}

func handleGetBlock(blk block.SignedBeaconBlock, err error) error {
	if invalidBlockIdErr, ok := err.(*blockIdParseError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid block ID: %v", invalidBlockIdErr)
	}
	if err != nil {
		return status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if blk == nil || blk.IsNil() {
		return status.Errorf(codes.Internal, "Could not find requested block")
	}
	return nil
}
