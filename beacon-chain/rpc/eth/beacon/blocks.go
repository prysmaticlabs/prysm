package beacon

import (
	"context"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	rpchelpers "github.com/prysmaticlabs/prysm/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/time/slots"
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

// GetWeakSubjectivity computes the starting epoch of the current weak subjectivity period, and then also
// determines the best block root and state root to use for a Checkpoint Sync starting from that point.
func (bs *Server) GetWeakSubjectivity(ctx context.Context, _ *empty.Empty) (*ethpbv1.WeakSubjectivityResponse, error) {
	if err := rpchelpers.ValidateSync(ctx, bs.SyncChecker, bs.HeadFetcher, bs.GenesisTimeFetcher); err != nil {
		// This is already a grpc error, so we can't wrap it any further
		return nil, err
	}

	hs, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not get head state")
	}
	wsEpoch, err := helpers.LatestWeakSubjectivityEpoch(ctx, hs, params.BeaconConfig())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get weak subjectivity epoch: %v", err)
	}
	wsSlot, err := slots.EpochStart(wsEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get weak subjectivity slot: %v", err)
	}
	cbr, cb, err := bs.CanonicalHistory.BlockForSlot(ctx, wsSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("could not find highest block below slot %d", wsSlot))
	}
	stateRoot := bytesutil.ToBytes32(cb.Block().StateRoot())
	log.Printf("weak subjectivity checkpoint reported as epoch=%d, block root=%#x, state root=%#x", wsEpoch, cbr, stateRoot)
	return &ethpbv1.WeakSubjectivityResponse{
		Data: &ethpbv1.WeakSubjectivityData{
			WsCheckpoint: &ethpbv1.Checkpoint{
				Epoch: wsEpoch,
				Root:  cbr[:],
			},
			StateRoot: stateRoot[:],
		},
	}, nil
}

// GetBlockHeader retrieves block header for given block id.
func (bs *Server) GetBlockHeader(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockHeaderResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockHeader")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockHeader")
	}
	v1alpha1Header, err := blk.Header()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block header from block: %v", err)
	}
	header := migration.V1Alpha1SignedHeaderToV1(v1alpha1Header)
	headerRoot, err := header.Message.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not hash block header: %v", err)
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not hash block: %v", err)
	}
	canonical, err := bs.ChainInfoFetcher.IsCanonical(ctx, blkRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if block root is canonical: %v", err)
	}
	isOptimistic, err := bs.HeadFetcher.IsOptimisticForRoot(ctx, blkRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if block is optimistic: %v", err)
	}

	return &ethpbv1.BlockHeaderResponse{
		Data: &ethpbv1.BlockHeaderContainer{
			Root:      headerRoot[:],
			Canonical: canonical,
			Header: &ethpbv1.BeaconBlockHeaderContainer{
				Message:   header.Message,
				Signature: header.Signature,
			},
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

// ListBlockHeaders retrieves block headers matching given query. By default it will fetch current head slot blocks.
func (bs *Server) ListBlockHeaders(ctx context.Context, req *ethpbv1.BlockHeadersRequest) (*ethpbv1.BlockHeadersResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListBlockHeaders")
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

	isOptimistic := false
	blkHdrs := make([]*ethpbv1.BlockHeaderContainer, len(blks))
	for i, bl := range blks {
		v1alpha1Header, err := bl.Header()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get block header from block: %v", err)
		}
		header := migration.V1Alpha1SignedHeaderToV1(v1alpha1Header)
		headerRoot, err := header.Message.HashTreeRoot()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not hash block header: %v", err)
		}
		canonical, err := bs.ChainInfoFetcher.IsCanonical(ctx, blkRoots[i])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not determine if block root is canonical: %v", err)
		}
		if !isOptimistic {
			isOptimistic, err = bs.HeadFetcher.IsOptimisticForRoot(ctx, blkRoots[i])
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not check if block is optimistic: %v", err)
			}
		}
		blkHdrs[i] = &ethpbv1.BlockHeaderContainer{
			Root:      headerRoot[:],
			Canonical: canonical,
			Header: &ethpbv1.BeaconBlockHeaderContainer{
				Message:   header.Message,
				Signature: header.Signature,
			},
		}
	}

	return &ethpbv1.BlockHeadersResponse{Data: blkHdrs, ExecutionOptimistic: isOptimistic}, nil
}

// SubmitBlock instructs the beacon node to broadcast a newly signed beacon block to the beacon network, to be
// included in the beacon chain. The beacon node is not required to validate the signed BeaconBlock, and a successful
// response (20X) only indicates that the broadcast has been successful. The beacon node is expected to integrate the
// new block into its state, and therefore validate the block internally, however blocks which fail the validation are
// still broadcast but a different status code is returned (202).
func (bs *Server) SubmitBlock(ctx context.Context, req *ethpbv2.SignedBeaconBlockContainerV2) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitBlock")
	defer span.End()

	var wsb block.SignedBeaconBlock
	var err error
	var v1alpha1Blk interface{}

	switch blk := req.Message.(type) {
	case *ethpbv2.SignedBeaconBlockContainerV2_Phase0Block:
		v1alpha1Blk, err = migration.V1ToV1Alpha1SignedBlock(&ethpbv1.SignedBeaconBlock{Block: blk.Phase0Block, Signature: req.Signature})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Could not convert block to v1 block: %v", err)
		}
		wsb, err = wrapper.WrappedSignedBeaconBlock(v1alpha1Blk)
	case *ethpbv2.SignedBeaconBlockContainerV2_AltairBlock:
		v1alpha1Blk, err = migration.AltairToV1Alpha1SignedBlock(&ethpbv2.SignedBeaconBlockAltair{Message: blk.AltairBlock, Signature: req.Signature})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Could not convert block to v1 block: %v", err)
		}
		wsb, err = wrapper.WrappedSignedBeaconBlock(v1alpha1Blk)
	case *ethpbv2.SignedBeaconBlockContainerV2_BellatrixBlock:
		v1alpha1Blk, err = migration.BellatrixToV1Alpha1SignedBlock(&ethpbv2.SignedBeaconBlockBellatrix{Message: blk.BellatrixBlock, Signature: req.Signature})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Could not convert block to v1 block: %v", err)
		}
		wsb, err = wrapper.WrappedSignedBeaconBlock(v1alpha1Blk)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unsupported block type %T", req.Message)
	}

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not wrap beacon block: %v", err)
	}

	root, err := wsb.Block().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not tree hash block: %v", err)
	}

	// Do not block proposal critical path with debug logging or block feed updates.
	defer func() {
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(root[:]))).Debugf(
			"Block proposal received via RPC")
		bs.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: &blockfeed.ReceivedBlockData{SignedBlock: wsb, IsOptimistic: false},
		})
	}()

	// Broadcast the new block to the network.
	if err := bs.Broadcaster.Broadcast(ctx, wsb.Proto()); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast block: %v", err)
	}

	if err := bs.BlockReceiver.ReceiveBlock(ctx, wsb, root); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process beacon block: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetBlock retrieves block details for given block ID.
func (bs *Server) GetBlock(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlock")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlock")
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
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockSSZ")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockSSZ")
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
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockV2")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockV2")
	}

	_, err = blk.PbPhase0Block()
	if err == nil {
		v1Blk, err := migration.SignedBeaconBlock(blk)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		return &ethpbv2.BlockResponseV2{
			Version: ethpbv2.Version_PHASE0,
			Data: &ethpbv2.SignedBeaconBlockContainerV2{
				Message:   &ethpbv2.SignedBeaconBlockContainerV2_Phase0Block{Phase0Block: v1Blk.Block},
				Signature: v1Blk.Signature,
			},
			ExecutionOptimistic: false,
		}, nil
	}
	// ErrUnsupportedPhase0Block means that we have another block type
	if !errors.Is(err, wrapper.ErrUnsupportedPhase0Block) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}

	altairBlk, err := blk.PbAltairBlock()
	if err == nil {
		if altairBlk == nil {
			return nil, status.Errorf(codes.Internal, "Nil block")
		}
		v2Blk, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlk.Block)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		return &ethpbv2.BlockResponseV2{
			Version: ethpbv2.Version_ALTAIR,
			Data: &ethpbv2.SignedBeaconBlockContainerV2{
				Message:   &ethpbv2.SignedBeaconBlockContainerV2_AltairBlock{AltairBlock: v2Blk},
				Signature: blk.Signature(),
			},
			ExecutionOptimistic: false,
		}, nil
	}
	// ErrUnsupportedAltairBlock means that we have another block type
	if !errors.Is(err, wrapper.ErrUnsupportedAltairBlock) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}

	bellatrixBlk, err := blk.PbBellatrixBlock()
	if err == nil {
		if bellatrixBlk == nil {
			return nil, status.Errorf(codes.Internal, "Nil block")
		}
		v2Blk, err := migration.V1Alpha1BeaconBlockBellatrixToV2(bellatrixBlk.Block)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		root, err := blk.Block().HashTreeRoot()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get block root: %v", err)
		}
		isOptimistic, err := bs.HeadFetcher.IsOptimisticForRoot(ctx, root)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not check if block is optimistic: %v", err)
		}
		return &ethpbv2.BlockResponseV2{
			Version: ethpbv2.Version_BELLATRIX,
			Data: &ethpbv2.SignedBeaconBlockContainerV2{
				Message:   &ethpbv2.SignedBeaconBlockContainerV2_BellatrixBlock{BellatrixBlock: v2Blk},
				Signature: blk.Signature(),
			},
			ExecutionOptimistic: isOptimistic,
		}, nil
	}
	// ErrUnsupportedBellatrixBlock means that we have another block type
	if !errors.Is(err, wrapper.ErrUnsupportedBellatrixBlock) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}

	return nil, status.Errorf(codes.Internal, "Unknown block type %T", blk)
}

// GetBlockSSZV2 returns the SSZ-serialized version of the beacon block for given block ID.
func (bs *Server) GetBlockSSZV2(ctx context.Context, req *ethpbv2.BlockRequestV2) (*ethpbv2.BlockSSZResponseV2, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockSSZV2")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockSSZV2")
	}

	_, err = blk.PbPhase0Block()
	if err == nil {
		signedBeaconBlock, err := migration.SignedBeaconBlock(blk)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		sszBlock, err := signedBeaconBlock.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ: %v", err)
		}
		return &ethpbv2.BlockSSZResponseV2{Version: ethpbv2.Version_PHASE0, Data: sszBlock}, nil
	}
	// ErrUnsupportedPhase0Block means that we have another block type
	if !errors.Is(err, wrapper.ErrUnsupportedPhase0Block) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}

	altairBlk, err := blk.PbAltairBlock()
	if err == nil {
		if altairBlk == nil {
			return nil, status.Errorf(codes.Internal, "Nil block")
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
		return &ethpbv2.BlockSSZResponseV2{Version: ethpbv2.Version_ALTAIR, Data: sszData}, nil
	}
	// ErrUnsupportedAltairBlock means that we have another block type
	if !errors.Is(err, wrapper.ErrUnsupportedAltairBlock) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}

	bellatrixBlk, err := blk.PbBellatrixBlock()
	if err == nil {
		if bellatrixBlk == nil {
			return nil, status.Errorf(codes.Internal, "Nil block")
		}
		v2Blk, err := migration.V1Alpha1BeaconBlockBellatrixToV2(bellatrixBlk.Block)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		data := &ethpbv2.SignedBeaconBlockBellatrix{
			Message:   v2Blk,
			Signature: blk.Signature(),
		}
		sszData, err := data.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ: %v", err)
		}
		return &ethpbv2.BlockSSZResponseV2{Version: ethpbv2.Version_BELLATRIX, Data: sszData}, nil
	}
	// ErrUnsupportedBellatrixBlock means that we have another block type
	if !errors.Is(err, wrapper.ErrUnsupportedBellatrixBlock) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}

	return nil, status.Errorf(codes.Internal, "Unknown block type %T", blk)
}

// GetBlockRoot retrieves hashTreeRoot of BeaconBlock/BeaconBlockHeader.
func (bs *Server) GetBlockRoot(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockRootResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockRoot")
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
		if err := helpers.BeaconBlockIsNil(blk); err != nil {
			return nil, status.Errorf(codes.NotFound, "Could not find genesis block: %v", err)
		}
		blkRoot, err := blk.Block().HashTreeRoot()
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not hash genesis block")
		}
		root = blkRoot[:]
	default:
		if len(req.BlockId) == 32 {
			blk, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(req.BlockId))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve block for block root %#x: %v", req.BlockId, err)
			}
			if err := helpers.BeaconBlockIsNil(blk); err != nil {
				return nil, status.Errorf(codes.NotFound, "Could not find block: %v", err)
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

	isOptimistic, err := bs.HeadFetcher.IsOptimisticForRoot(ctx, bytesutil.ToBytes32(root))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if block is optimistic: %v", err)
	}

	return &ethpbv1.BlockRootResponse{
		Data: &ethpbv1.BlockRootContainer{
			Root: root,
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

// ListBlockAttestations retrieves attestation included in requested block.
func (bs *Server) ListBlockAttestations(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockAttestationsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListBlockAttestations")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, err
	}

	_, err = blk.PbPhase0Block()
	if err != nil && !errors.Is(err, wrapper.ErrUnsupportedPhase0Block) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	if err == nil {
		v1Blk, err := migration.SignedBeaconBlock(blk)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		return &ethpbv1.BlockAttestationsResponse{
			Data:                v1Blk.Block.Body.Attestations,
			ExecutionOptimistic: false,
		}, nil
	}

	altairBlk, err := blk.PbAltairBlock()
	if err != nil && !errors.Is(err, wrapper.ErrUnsupportedAltairBlock) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	if err == nil {
		if altairBlk == nil {
			return nil, status.Errorf(codes.Internal, "Nil block")
		}
		v2Blk, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlk.Block)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		return &ethpbv1.BlockAttestationsResponse{
			Data:                v2Blk.Body.Attestations,
			ExecutionOptimistic: false,
		}, nil
	}

	bellatrixBlock, err := blk.PbBellatrixBlock()
	if err != nil && !errors.Is(err, wrapper.ErrUnsupportedBellatrixBlock) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	if err == nil {
		if bellatrixBlock == nil {
			return nil, status.Errorf(codes.Internal, "Nil block")
		}
		v2Blk, err := migration.V1Alpha1BeaconBlockBellatrixToV2(bellatrixBlock.Block)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
		}
		root, err := blk.Block().HashTreeRoot()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get block root: %v", err)
		}
		isOptimistic, err := bs.HeadFetcher.IsOptimisticForRoot(ctx, root)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not check if block is optimistic: %v", err)
		}
		return &ethpbv1.BlockAttestationsResponse{
			Data:                v2Blk.Body.Attestations,
			ExecutionOptimistic: isOptimistic,
		}, nil
	}

	return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
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
			for i, b := range blks {
				canonical, err := bs.ChainInfoFetcher.IsCanonical(ctx, roots[i])
				if err != nil {
					return nil, status.Errorf(codes.Internal, "Could not determine if block root is canonical: %v", err)
				}
				if canonical {
					blk = b
					break
				}
			}
		}
	}
	return blk, nil
}

func handleGetBlockError(blk block.SignedBeaconBlock, err error) error {
	if invalidBlockIdErr, ok := err.(*blockIdParseError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid block ID: %v", invalidBlockIdErr)
	}
	if err != nil {
		return status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if err := helpers.BeaconBlockIsNil(blk); err != nil {
		return status.Errorf(codes.NotFound, "Could not find requested block: %v", err)
	}
	return nil
}
