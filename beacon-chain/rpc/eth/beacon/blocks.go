package beacon

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filters"
	rpchelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	consensus_types "github.com/prysmaticlabs/prysm/v4/consensus-types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz/detect"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	errNilBlock = errors.New("nil block")
)

// GetWeakSubjectivity computes the starting epoch of the current weak subjectivity period, and then also
// determines the best block root and state root to use for a Checkpoint Sync starting from that point.
// DEPRECATED: GetWeakSubjectivity endpoint will no longer be supported
func (bs *Server) GetWeakSubjectivity(ctx context.Context, _ *empty.Empty) (*ethpbv1.WeakSubjectivityResponse, error) {
	if err := rpchelpers.ValidateSyncGRPC(ctx, bs.SyncChecker, bs.HeadFetcher, bs.GenesisTimeFetcher, bs.OptimisticModeFetcher); err != nil {
		// This is already a grpc error, so we can't wrap it any further
		return nil, err
	}

	hs, err := bs.HeadFetcher.HeadStateReadOnly(ctx)
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
	cbr, err := bs.CanonicalHistory.BlockRootForSlot(ctx, wsSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("could not find highest block below slot %d", wsSlot))
	}
	cb, err := bs.BeaconDB.Block(ctx, cbr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("block with root %#x from slot index %d not found in db", cbr, wsSlot))
	}
	stateRoot := cb.Block().StateRoot()
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

	blk, err := bs.Blocker.Block(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, err
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
	isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, blkRoot)
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
		Finalized:           bs.FinalizationFetcher.IsFinalized(ctx, blkRoot),
	}, nil
}

// ListBlockHeaders retrieves block headers matching given query. By default it will fetch current head slot blocks.
func (bs *Server) ListBlockHeaders(ctx context.Context, req *ethpbv1.BlockHeadersRequest) (*ethpbv1.BlockHeadersResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListBlockHeaders")
	defer span.End()

	var err error
	var blks []interfaces.ReadOnlySignedBeaconBlock
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
		blks, err = bs.BeaconDB.BlocksBySlot(ctx, slot)
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
	isFinalized := true
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
			isOptimistic, err = bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, blkRoots[i])
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not check if block is optimistic: %v", err)
			}
		}
		if isFinalized {
			isFinalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoots[i])
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

	return &ethpbv1.BlockHeadersResponse{Data: blkHdrs, ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
}

// SubmitBlock instructs the beacon node to broadcast a newly signed beacon block to the beacon network, to be
// included in the beacon chain. The beacon node is not required to validate the signed ReadOnlyBeaconBlock, and a successful
// response (20X) only indicates that the broadcast has been successful. The beacon node is expected to integrate the
// new block into its state, and therefore validate the block internally, however blocks which fail the validation are
// still broadcast but a different status code is returned (202).
func (bs *Server) SubmitBlock(ctx context.Context, req *ethpbv2.SignedBeaconBlockContainer) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitBlock")
	defer span.End()

	if err := rpchelpers.ValidateSyncGRPC(ctx, bs.SyncChecker, bs.HeadFetcher, bs.TimeFetcher, bs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	switch blkContainer := req.Message.(type) {
	case *ethpbv2.SignedBeaconBlockContainer_Phase0Block:
		if err := bs.submitPhase0Block(ctx, blkContainer.Phase0Block, req.Signature); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBeaconBlockContainer_AltairBlock:
		if err := bs.submitAltairBlock(ctx, blkContainer.AltairBlock, req.Signature); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBeaconBlockContainer_BellatrixBlock:
		if err := bs.submitBellatrixBlock(ctx, blkContainer.BellatrixBlock, req.Signature); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBeaconBlockContainer_CapellaBlock:
		if err := bs.submitCapellaBlock(ctx, blkContainer.CapellaBlock, req.Signature); err != nil {
			return nil, err
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unsupported block container type %T", blkContainer)
	}

	return &emptypb.Empty{}, nil
}

// SubmitBlockSSZ instructs the beacon node to broadcast a newly signed beacon block to the beacon network, to be
// included in the beacon chain. The beacon node is not required to validate the signed ReadOnlyBeaconBlock, and a successful
// response (20X) only indicates that the broadcast has been successful. The beacon node is expected to integrate the
// new block into its state, and therefore validate the block internally, however blocks which fail the validation are
// still broadcast but a different status code is returned (202).
//
// The provided block must be SSZ-serialized.
func (bs *Server) SubmitBlockSSZ(ctx context.Context, req *ethpbv2.SSZContainer) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitBlockSSZ")
	defer span.End()

	if err := rpchelpers.ValidateSyncGRPC(ctx, bs.SyncChecker, bs.HeadFetcher, bs.TimeFetcher, bs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not read "+api.VersionHeader+" header")
	}
	ver := md.Get(api.VersionHeader)
	if len(ver) == 0 {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not read "+api.VersionHeader+" header")
	}
	schedule := forks.NewOrderedSchedule(params.BeaconConfig())
	forkVer, err := schedule.VersionForName(ver[0])
	if err != nil {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not determine fork version: %v", err)
	}
	unmarshaler, err := detect.FromForkVersion(forkVer)
	if err != nil {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not create unmarshaler: %v", err)
	}
	block, err := unmarshaler.UnmarshalBeaconBlock(req.Data)
	if err != nil {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not unmarshal request data into block: %v", err)
	}

	switch forkVer {
	case bytesutil.ToBytes4(params.BeaconConfig().CapellaForkVersion):
		if block.IsBlinded() {
			return nil, status.Error(codes.InvalidArgument, "Submitted block is blinded")
		}
		b, err := block.PbCapellaBlock()
		if err != nil {
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not get proto block: %v", err)
		}
		_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Capella{
				Capella: b,
			},
		})
		if err != nil {
			if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
				return &emptypb.Empty{}, status.Error(codes.InvalidArgument, err.Error())
			}
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not propose block: %v", err)
		}
		return &emptypb.Empty{}, nil
	case bytesutil.ToBytes4(params.BeaconConfig().BellatrixForkVersion):
		if block.IsBlinded() {
			return nil, status.Error(codes.InvalidArgument, "Submitted block is blinded")
		}
		b, err := block.PbBellatrixBlock()
		if err != nil {
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not get proto block: %v", err)
		}
		_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Bellatrix{
				Bellatrix: b,
			},
		})
		if err != nil {
			if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
				return &emptypb.Empty{}, status.Error(codes.InvalidArgument, err.Error())
			}
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not propose block: %v", err)
		}
		return &emptypb.Empty{}, nil
	case bytesutil.ToBytes4(params.BeaconConfig().AltairForkVersion):
		b, err := block.PbAltairBlock()
		if err != nil {
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not get proto block: %v", err)
		}
		_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Altair{
				Altair: b,
			},
		})
		if err != nil {
			if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
				return &emptypb.Empty{}, status.Error(codes.InvalidArgument, err.Error())
			}
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not propose block: %v", err)
		}
		return &emptypb.Empty{}, nil
	case bytesutil.ToBytes4(params.BeaconConfig().GenesisForkVersion):
		b, err := block.PbPhase0Block()
		if err != nil {
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not get proto block: %v", err)
		}
		_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Phase0{
				Phase0: b,
			},
		})
		if err != nil {
			if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
				return &emptypb.Empty{}, status.Error(codes.InvalidArgument, err.Error())
			}
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not propose block: %v", err)
		}
		return &emptypb.Empty{}, nil
	default:
		return &emptypb.Empty{}, status.Errorf(codes.InvalidArgument, "Unsupported fork %s", string(forkVer[:]))
	}
}

// GetBlock retrieves block details for given block ID.
// DEPRECATED: please use GetBlockV2 instead
func (bs *Server) GetBlock(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlock")
	defer span.End()

	blk, err := bs.Blocker.Block(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
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
// DEPRECATED: please use GetBlockV2SSZ instead
func (bs *Server) GetBlockSSZ(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockSSZResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockSSZ")
	defer span.End()

	blk, err := bs.Blocker.Block(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
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
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockV2")
	defer span.End()

	blk, err := bs.Blocker.Block(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, err
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}

	result, err := getBlockPhase0(blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	if err := grpc.SetHeader(ctx, metadata.Pairs(api.VersionHeader, version.String(blk.Version()))); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not set "+api.VersionHeader+" header: %v", err)
	}
	result, err = getBlockAltair(blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = bs.getBlockBellatrix(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = bs.getBlockCapella(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	return nil, status.Errorf(codes.Internal, "Unknown block type %T", blk)
}

// GetBlockSSZV2 returns the SSZ-serialized version of the beacon block for given block ID.
func (bs *Server) GetBlockSSZV2(ctx context.Context, req *ethpbv2.BlockRequestV2) (*ethpbv2.SSZContainer, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlockSSZV2")
	defer span.End()

	blk, err := bs.Blocker.Block(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, err
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}

	result, err := getSSZBlockPhase0(blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = getSSZBlockAltair(blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = bs.getSSZBlockBellatrix(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = bs.getSSZBlockCapella(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}

	return nil, status.Errorf(codes.Internal, "Unknown block type %T", blk)
}

// ListBlockAttestations retrieves attestation included in requested block.
func (bs *Server) ListBlockAttestations(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv1.BlockAttestationsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListBlockAttestations")
	defer span.End()

	blk, err := bs.Blocker.Block(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, err
	}

	v1Alpha1Attestations := blk.Block().Body().Attestations()
	v1Attestations := make([]*ethpbv1.Attestation, 0, len(v1Alpha1Attestations))
	for _, att := range v1Alpha1Attestations {
		migratedAtt := migration.V1Alpha1AttestationToV1(att)
		v1Attestations = append(v1Attestations, migratedAtt)
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block root: %v", err)
	}
	isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if block is optimistic: %v", err)
	}
	return &ethpbv1.BlockAttestationsResponse{
		Data:                v1Attestations,
		ExecutionOptimistic: isOptimistic,
		Finalized:           bs.FinalizationFetcher.IsFinalized(ctx, root),
	}, nil
}

func handleGetBlockError(blk interfaces.ReadOnlySignedBeaconBlock, err error) error {
	if invalidBlockIdErr, ok := err.(*lookup.BlockIdParseError); ok {
		return status.Errorf(codes.InvalidArgument, "Invalid block ID: %v", invalidBlockIdErr)
	}
	if err != nil {
		return status.Errorf(codes.Internal, "Could not get block from block ID: %v", err)
	}
	if err := blocks.BeaconBlockIsNil(blk); err != nil {
		return status.Errorf(codes.NotFound, "Could not find requested block: %v", err)
	}
	return nil
}

func getBlockPhase0(blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.BlockResponseV2, error) {
	phase0Blk, err := blk.PbPhase0Block()
	if err != nil {
		return nil, err
	}
	if phase0Blk == nil {
		return nil, errNilBlock
	}
	v1Blk, err := migration.SignedBeaconBlock(blk)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	return &ethpbv2.BlockResponseV2{
		Version: ethpbv2.Version_PHASE0,
		Data: &ethpbv2.SignedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBeaconBlockContainer_Phase0Block{Phase0Block: v1Blk.Block},
			Signature: v1Blk.Signature,
		},
		ExecutionOptimistic: false,
	}, nil
}

func getBlockAltair(blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.BlockResponseV2, error) {
	altairBlk, err := blk.PbAltairBlock()
	if err != nil {
		return nil, err
	}
	if altairBlk == nil {
		return nil, errNilBlock
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlk.Block)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	sig := blk.Signature()
	return &ethpbv2.BlockResponseV2{
		Version: ethpbv2.Version_ALTAIR,
		Data: &ethpbv2.SignedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBeaconBlockContainer_AltairBlock{AltairBlock: v2Blk},
			Signature: sig[:],
		},
		ExecutionOptimistic: false,
	}, nil
}

func (bs *Server) getBlockBellatrix(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.BlockResponseV2, error) {
	bellatrixBlk, err := blk.PbBellatrixBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedBellatrixBlk, err := blk.PbBlindedBellatrixBlock(); err == nil {
				if blindedBellatrixBlk == nil {
					return nil, errNilBlock
				}
				signedFullBlock, err := bs.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
				if err != nil {
					return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
				}
				bellatrixBlk, err = signedFullBlock.PbBellatrixBlock()
				if err != nil {
					return nil, errors.Wrapf(err, "could not get signed beacon block")
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockBellatrixToV2(bellatrixBlk.Block)
				if err != nil {
					return nil, errors.Wrapf(err, "could not convert beacon block")
				}
				root, err := blk.Block().HashTreeRoot()
				if err != nil {
					return nil, errors.Wrapf(err, "could not get block root")
				}
				isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
				if err != nil {
					return nil, errors.Wrapf(err, "could not check if block is optimistic")
				}
				sig := blk.Signature()
				return &ethpbv2.BlockResponseV2{
					Version: ethpbv2.Version_BELLATRIX,
					Data: &ethpbv2.SignedBeaconBlockContainer{
						Message:   &ethpbv2.SignedBeaconBlockContainer_BellatrixBlock{BellatrixBlock: v2Blk},
						Signature: sig[:],
					},
					ExecutionOptimistic: isOptimistic,
				}, nil
			}
			return nil, err
		}
		return nil, err
	}

	if bellatrixBlk == nil {
		return nil, errNilBlock
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockBellatrixToV2(bellatrixBlk.Block)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert beacon block")
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	sig := blk.Signature()
	return &ethpbv2.BlockResponseV2{
		Version: ethpbv2.Version_BELLATRIX,
		Data: &ethpbv2.SignedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBeaconBlockContainer_BellatrixBlock{BellatrixBlock: v2Blk},
			Signature: sig[:],
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

func (bs *Server) getBlockCapella(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.BlockResponseV2, error) {
	capellaBlk, err := blk.PbCapellaBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedCapellaBlk, err := blk.PbBlindedCapellaBlock(); err == nil {
				if blindedCapellaBlk == nil {
					return nil, errNilBlock
				}
				signedFullBlock, err := bs.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
				if err != nil {
					return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
				}
				capellaBlk, err = signedFullBlock.PbCapellaBlock()
				if err != nil {
					return nil, errors.Wrapf(err, "could not get signed beacon block")
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockCapellaToV2(capellaBlk.Block)
				if err != nil {
					return nil, errors.Wrapf(err, "could not convert beacon block")
				}
				root, err := blk.Block().HashTreeRoot()
				if err != nil {
					return nil, errors.Wrapf(err, "could not get block root")
				}
				isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
				if err != nil {
					return nil, errors.Wrapf(err, "could not check if block is optimistic")
				}
				sig := blk.Signature()
				return &ethpbv2.BlockResponseV2{
					Version: ethpbv2.Version_CAPELLA,
					Data: &ethpbv2.SignedBeaconBlockContainer{
						Message:   &ethpbv2.SignedBeaconBlockContainer_CapellaBlock{CapellaBlock: v2Blk},
						Signature: sig[:],
					},
					ExecutionOptimistic: isOptimistic,
				}, nil
			}
			return nil, err
		}
		return nil, err
	}

	if capellaBlk == nil {
		return nil, errNilBlock
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockCapellaToV2(capellaBlk.Block)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert beacon block")
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	sig := blk.Signature()
	return &ethpbv2.BlockResponseV2{
		Version: ethpbv2.Version_CAPELLA,
		Data: &ethpbv2.SignedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBeaconBlockContainer_CapellaBlock{CapellaBlock: v2Blk},
			Signature: sig[:],
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

func getSSZBlockPhase0(blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.SSZContainer, error) {
	phase0Blk, err := blk.PbPhase0Block()
	if err != nil {
		return nil, err
	}
	if phase0Blk == nil {
		return nil, errNilBlock
	}
	signedBeaconBlock, err := migration.SignedBeaconBlock(blk)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	sszBlock, err := signedBeaconBlock.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return &ethpbv2.SSZContainer{Version: ethpbv2.Version_PHASE0, ExecutionOptimistic: false, Data: sszBlock}, nil
}

func getSSZBlockAltair(blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.SSZContainer, error) {
	altairBlk, err := blk.PbAltairBlock()
	if err != nil {
		return nil, err
	}
	if altairBlk == nil {
		return nil, errNilBlock
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlk.Block)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	sig := blk.Signature()
	data := &ethpbv2.SignedBeaconBlockAltair{
		Message:   v2Blk,
		Signature: sig[:],
	}
	sszData, err := data.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return &ethpbv2.SSZContainer{Version: ethpbv2.Version_ALTAIR, ExecutionOptimistic: false, Data: sszData}, nil
}

func (bs *Server) getSSZBlockBellatrix(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.SSZContainer, error) {
	bellatrixBlk, err := blk.PbBellatrixBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedBellatrixBlk, err := blk.PbBlindedBellatrixBlock(); err == nil {
				if blindedBellatrixBlk == nil {
					return nil, errNilBlock
				}
				signedFullBlock, err := bs.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
				if err != nil {
					return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
				}
				bellatrixBlk, err = signedFullBlock.PbBellatrixBlock()
				if err != nil {
					return nil, errors.Wrapf(err, "could not get signed beacon block")
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockBellatrixToV2(bellatrixBlk.Block)
				if err != nil {
					return nil, errors.Wrapf(err, "could not convert signed beacon block")
				}
				root, err := blk.Block().HashTreeRoot()
				if err != nil {
					return nil, errors.Wrapf(err, "could not get block root")
				}
				isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
				if err != nil {
					return nil, errors.Wrapf(err, "could not check if block is optimistic")
				}
				sig := blk.Signature()
				data := &ethpbv2.SignedBeaconBlockBellatrix{
					Message:   v2Blk,
					Signature: sig[:],
				}
				sszData, err := data.MarshalSSZ()
				if err != nil {
					return nil, errors.Wrapf(err, "could not marshal block into SSZ")
				}
				return &ethpbv2.SSZContainer{
					Version:             ethpbv2.Version_BELLATRIX,
					ExecutionOptimistic: isOptimistic,
					Data:                sszData,
				}, nil
			}
			return nil, err
		}
		return nil, err
	}

	if bellatrixBlk == nil {
		return nil, errNilBlock
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockBellatrixToV2(bellatrixBlk.Block)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert signed beacon block")
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	sig := blk.Signature()
	data := &ethpbv2.SignedBeaconBlockBellatrix{
		Message:   v2Blk,
		Signature: sig[:],
	}
	sszData, err := data.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return &ethpbv2.SSZContainer{Version: ethpbv2.Version_BELLATRIX, ExecutionOptimistic: isOptimistic, Data: sszData}, nil
}

func (bs *Server) getSSZBlockCapella(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.SSZContainer, error) {
	capellaBlk, err := blk.PbCapellaBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedCapellaBlk, err := blk.PbBlindedCapellaBlock(); err == nil {
				if blindedCapellaBlk == nil {
					return nil, errNilBlock
				}
				signedFullBlock, err := bs.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
				if err != nil {
					return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
				}
				capellaBlk, err = signedFullBlock.PbCapellaBlock()
				if err != nil {
					return nil, errors.Wrapf(err, "could not get signed beacon block")
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockCapellaToV2(capellaBlk.Block)
				if err != nil {
					return nil, errors.Wrapf(err, "could not convert signed beacon block")
				}
				root, err := blk.Block().HashTreeRoot()
				if err != nil {
					return nil, errors.Wrapf(err, "could not get block root")
				}
				isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
				if err != nil {
					return nil, errors.Wrapf(err, "could not check if block is optimistic")
				}
				sig := blk.Signature()
				data := &ethpbv2.SignedBeaconBlockCapella{
					Message:   v2Blk,
					Signature: sig[:],
				}
				sszData, err := data.MarshalSSZ()
				if err != nil {
					return nil, errors.Wrapf(err, "could not marshal block into SSZ")
				}
				return &ethpbv2.SSZContainer{
					Version:             ethpbv2.Version_CAPELLA,
					ExecutionOptimistic: isOptimistic,
					Data:                sszData,
				}, nil
			}
			return nil, err
		}
		return nil, err
	}

	if capellaBlk == nil {
		return nil, errNilBlock
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockCapellaToV2(capellaBlk.Block)
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert signed beacon block")
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := bs.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	sig := blk.Signature()
	data := &ethpbv2.SignedBeaconBlockCapella{
		Message:   v2Blk,
		Signature: sig[:],
	}
	sszData, err := data.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return &ethpbv2.SSZContainer{Version: ethpbv2.Version_CAPELLA, ExecutionOptimistic: isOptimistic, Data: sszData}, nil
}

func (bs *Server) submitPhase0Block(ctx context.Context, phase0Blk *ethpbv1.BeaconBlock, sig []byte) error {
	v1alpha1Blk, err := migration.V1ToV1Alpha1SignedBlock(&ethpbv1.SignedBeaconBlock{Block: phase0Blk, Signature: sig})
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "Could not convert block: %v", err)
	}
	_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Phase0{
			Phase0: v1alpha1Blk,
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Errorf(codes.Internal, "Could not propose block: %v", err)
	}
	return nil
}

func (bs *Server) submitAltairBlock(ctx context.Context, altairBlk *ethpbv2.BeaconBlockAltair, sig []byte) error {
	v1alpha1Blk, err := migration.AltairToV1Alpha1SignedBlock(&ethpbv2.SignedBeaconBlockAltair{Message: altairBlk, Signature: sig})
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "Could not convert block %v", err)
	}
	_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Altair{
			Altair: v1alpha1Blk,
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Errorf(codes.Internal, "Could not propose block: %v", err)
	}
	return nil
}

func (bs *Server) submitBellatrixBlock(ctx context.Context, bellatrixBlk *ethpbv2.BeaconBlockBellatrix, sig []byte) error {
	v1alpha1Blk, err := migration.BellatrixToV1Alpha1SignedBlock(&ethpbv2.SignedBeaconBlockBellatrix{Message: bellatrixBlk, Signature: sig})
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "Could not convert block to v1 block")
	}
	_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Bellatrix{
			Bellatrix: v1alpha1Blk,
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Errorf(codes.Internal, "Could not propose block: %v", err)
	}
	return nil
}

func (bs *Server) submitCapellaBlock(ctx context.Context, capellaBlk *ethpbv2.BeaconBlockCapella, sig []byte) error {
	v1alpha1Blk, err := migration.CapellaToV1Alpha1SignedBlock(&ethpbv2.SignedBeaconBlockCapella{Message: capellaBlk, Signature: sig})
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "Could not convert block to v1 block")
	}
	_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_Capella{
			Capella: v1alpha1Blk,
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Errorf(codes.Internal, "Could not propose block: %v", err)
	}
	return nil
}
