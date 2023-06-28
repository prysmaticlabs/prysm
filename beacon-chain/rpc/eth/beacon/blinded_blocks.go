package beacon

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
	rpchelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	consensus_types "github.com/prysmaticlabs/prysm/v4/consensus-types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz/detect"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetBlindedBlock retrieves blinded block for given block id.
func (bs *Server) GetBlindedBlock(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv2.BlindedBlockResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlindedBlock")
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
	if err := grpc.SetHeader(ctx, metadata.Pairs(api.VersionHeader, version.String(blk.Version()))); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not set "+api.VersionHeader+" header: %v", err)
	}
	result, err := getBlindedBlockPhase0(blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = getBlindedBlockAltair(blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = bs.getBlindedBlockBellatrix(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = bs.getBlindedBlockCapella(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = bs.getBlindedBlockDeneb(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}

	return nil, status.Errorf(codes.Internal, "Unknown block type %T", blk)
}

// GetBlindedBlockSSZ returns the SSZ-serialized version of the blinded beacon block for given block id.
func (bs *Server) GetBlindedBlockSSZ(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv2.SSZContainer, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlindedBlockSSZ")
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
	result, err = bs.getBlindedSSZBlockBellatrix(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = bs.getBlindedSSZBlockCapella(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedField means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = bs.getBlindedSSZBlockDeneb(ctx, blk)
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

// SubmitBlindedBlock instructs the beacon node to use the components of the `SignedBlindedBeaconBlock` to construct
// and publish a `ReadOnlySignedBeaconBlock` by swapping out the `transactions_root` for the corresponding full list of `transactions`.
// The beacon node should broadcast a newly constructed `ReadOnlySignedBeaconBlock` to the beacon network,
// to be included in the beacon chain. The beacon node is not required to validate the signed
// `ReadOnlyBeaconBlock`, and a successful response (20X) only indicates that the broadcast has been
// successful. The beacon node is expected to integrate the new block into its state, and
// therefore validate the block internally, however blocks which fail the validation are still
// broadcast but a different status code is returned (202).
func (bs *Server) SubmitBlindedBlock(ctx context.Context, req *ethpbv2.SignedBlindedBeaconBlockContentsContainer) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitBlindedBlock")
	defer span.End()

	if err := rpchelpers.ValidateSyncGRPC(ctx, bs.SyncChecker, bs.HeadFetcher, bs.TimeFetcher, bs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	switch blkContainer := req.Message.(type) {
	case *ethpbv2.SignedBlindedBeaconBlockContentsContainer_DenebContents:
		if err := bs.submitBlindedDenebContents(ctx, blkContainer.DenebContents); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBlindedBeaconBlockContentsContainer_CapellaBlock:
		if err := bs.submitBlindedCapellaBlock(ctx, blkContainer.CapellaBlock.Message, blkContainer.CapellaBlock.Signature); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBlindedBeaconBlockContentsContainer_BellatrixBlock:
		if err := bs.submitBlindedBellatrixBlock(ctx, blkContainer.BellatrixBlock.Message, blkContainer.BellatrixBlock.Signature); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBlindedBeaconBlockContentsContainer_Phase0Block:
		if err := bs.submitPhase0Block(ctx, blkContainer.Phase0Block.Block, blkContainer.Phase0Block.Signature); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBlindedBeaconBlockContentsContainer_AltairBlock:
		if err := bs.submitAltairBlock(ctx, blkContainer.AltairBlock.Message, blkContainer.AltairBlock.Signature); err != nil {
			return nil, err
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unsupported block container type %T", blkContainer)
	}

	return &emptypb.Empty{}, nil
}

// SubmitBlindedBlockSSZ instructs the beacon node to use the components of the `SignedBlindedBeaconBlock` to construct
// and publish a `ReadOnlySignedBeaconBlock` by swapping out the `transactions_root` for the corresponding full list of `transactions`.
// The beacon node should broadcast a newly constructed `ReadOnlySignedBeaconBlock` to the beacon network,
// to be included in the beacon chain. The beacon node is not required to validate the signed
// `ReadOnlyBeaconBlock`, and a successful response (20X) only indicates that the broadcast has been
// successful. The beacon node is expected to integrate the new block into its state, and
// therefore validate the block internally, however blocks which fail the validation are still
// broadcast but a different status code is returned (202).
//
// The provided block must be SSZ-serialized.
func (bs *Server) SubmitBlindedBlockSSZ(ctx context.Context, req *ethpbv2.SSZContainer) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitBlindedBlockSSZ")
	defer span.End()

	if err := rpchelpers.ValidateSyncGRPC(ctx, bs.SyncChecker, bs.HeadFetcher, bs.TimeFetcher, bs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not read"+api.VersionHeader+" header")
	}
	ver := md.Get(api.VersionHeader)
	if len(ver) == 0 {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not read"+api.VersionHeader+" header")
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
	block, err := unmarshaler.UnmarshalBlindedBeaconBlock(req.Data)
	if err != nil {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not unmarshal request data into block: %v", err)
	}

	switch forkVer {
	case bytesutil.ToBytes4(params.BeaconConfig().CapellaForkVersion):
		if !block.IsBlinded() {
			return nil, status.Error(codes.InvalidArgument, "Submitted block is not blinded")
		}
		b, err := block.PbBlindedCapellaBlock()
		if err != nil {
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not get proto block: %v", err)
		}
		_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_BlindedCapella{
				BlindedCapella: b,
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
		if !block.IsBlinded() {
			return nil, status.Error(codes.InvalidArgument, "Submitted block is not blinded")
		}
		b, err := block.PbBlindedBellatrixBlock()
		if err != nil {
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not get proto block: %v", err)
		}
		_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{
				BlindedBellatrix: b,
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

func getBlindedBlockPhase0(blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.BlindedBlockResponse, error) {
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
	return &ethpbv2.BlindedBlockResponse{
		Version: ethpbv2.Version_PHASE0,
		Data: &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_Phase0Block{Phase0Block: v1Blk.Block},
			Signature: v1Blk.Signature,
		},
		ExecutionOptimistic: false,
	}, nil
}

func getBlindedBlockAltair(blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.BlindedBlockResponse, error) {
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
	return &ethpbv2.BlindedBlockResponse{
		Version: ethpbv2.Version_ALTAIR,
		Data: &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_AltairBlock{AltairBlock: v2Blk},
			Signature: sig[:],
		},
		ExecutionOptimistic: false,
	}, nil
}

func (bs *Server) getBlindedBlockBellatrix(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.BlindedBlockResponse, error) {
	bellatrixBlk, err := blk.PbBellatrixBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedBellatrixBlk, err := blk.PbBlindedBellatrixBlock(); err == nil {
				if blindedBellatrixBlk == nil {
					return nil, errNilBlock
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(blindedBellatrixBlk.Block)
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
				return &ethpbv2.BlindedBlockResponse{
					Version: ethpbv2.Version_BELLATRIX,
					Data: &ethpbv2.SignedBlindedBeaconBlockContainer{
						Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_BellatrixBlock{BellatrixBlock: v2Blk},
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
	blindedBlkInterface, err := blk.ToBlinded()
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert block to blinded block")
	}
	blindedBellatrixBlock, err := blindedBlkInterface.PbBlindedBellatrixBlock()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(blindedBellatrixBlock.Block)
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
	return &ethpbv2.BlindedBlockResponse{
		Version: ethpbv2.Version_BELLATRIX,
		Data: &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_BellatrixBlock{BellatrixBlock: v2Blk},
			Signature: sig[:],
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

func (bs *Server) getBlindedBlockCapella(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.BlindedBlockResponse, error) {
	capellaBlk, err := blk.PbCapellaBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedCapellaBlk, err := blk.PbBlindedCapellaBlock(); err == nil {
				if blindedCapellaBlk == nil {
					return nil, errNilBlock
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(blindedCapellaBlk.Block)
				if err != nil {
					return nil, errors.Wrapf(err, "Could not convert beacon block")
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
				return &ethpbv2.BlindedBlockResponse{
					Version: ethpbv2.Version_CAPELLA,
					Data: &ethpbv2.SignedBlindedBeaconBlockContainer{
						Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_CapellaBlock{CapellaBlock: v2Blk},
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
	blindedBlkInterface, err := blk.ToBlinded()
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert block to blinded block")
	}
	blindedCapellaBlock, err := blindedBlkInterface.PbBlindedCapellaBlock()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(blindedCapellaBlock.Block)
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
	return &ethpbv2.BlindedBlockResponse{
		Version: ethpbv2.Version_CAPELLA,
		Data: &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_CapellaBlock{CapellaBlock: v2Blk},
			Signature: sig[:],
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

func (bs *Server) getBlindedBlockDeneb(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.BlindedBlockResponse, error) {
	denebBlk, err := blk.PbDenebBlock()
	if err != nil {
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedDenebBlk, err := blk.PbBlindedDenebBlock(); err == nil {
				if blindedDenebBlk == nil {
					return nil, errNilBlock
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockBlindedDenebToV2Blinded(blindedDenebBlk.Block)
				if err != nil {
					return nil, errors.Wrapf(err, "Could not convert beacon block")
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
				return &ethpbv2.BlindedBlockResponse{
					Version: ethpbv2.Version_DENEB,
					Data: &ethpbv2.SignedBlindedBeaconBlockContainer{
						Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_DenebBlock{DenebBlock: v2Blk},
						Signature: sig[:],
					},
					ExecutionOptimistic: isOptimistic,
				}, nil
			}
			return nil, err
		}
		return nil, err
	}

	if denebBlk == nil {
		return nil, errNilBlock
	}
	blindedBlkInterface, err := blk.ToBlinded()
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert block to blinded block")
	}
	blindedDenebBlock, err := blindedBlkInterface.PbBlindedDenebBlock()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockBlindedDenebToV2Blinded(blindedDenebBlock.Block)
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
	return &ethpbv2.BlindedBlockResponse{
		Version: ethpbv2.Version_CAPELLA,
		Data: &ethpbv2.SignedBlindedBeaconBlockContainer{
			Message:   &ethpbv2.SignedBlindedBeaconBlockContainer_DenebBlock{DenebBlock: v2Blk},
			Signature: sig[:],
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

func (bs *Server) getBlindedSSZBlockBellatrix(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.SSZContainer, error) {
	bellatrixBlk, err := blk.PbBellatrixBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedBellatrixBlk, err := blk.PbBlindedBellatrixBlock(); err == nil {
				if blindedBellatrixBlk == nil {
					return nil, errNilBlock
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(blindedBellatrixBlk.Block)
				if err != nil {
					return nil, errors.Wrapf(err, "could not get signed beacon block")
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
				data := &ethpbv2.SignedBlindedBeaconBlockBellatrix{
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
	}

	if bellatrixBlk == nil {
		return nil, errNilBlock
	}
	blindedBlkInterface, err := blk.ToBlinded()
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert block to blinded block")
	}
	blindedBellatrixBlock, err := blindedBlkInterface.PbBlindedBellatrixBlock()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(blindedBellatrixBlock.Block)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
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
	data := &ethpbv2.SignedBlindedBeaconBlockBellatrix{
		Message:   v2Blk,
		Signature: sig[:],
	}
	sszData, err := data.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return &ethpbv2.SSZContainer{Version: ethpbv2.Version_BELLATRIX, ExecutionOptimistic: isOptimistic, Data: sszData}, nil
}

func (bs *Server) getBlindedSSZBlockCapella(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.SSZContainer, error) {
	capellaBlk, err := blk.PbCapellaBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedCapellaBlk, err := blk.PbBlindedCapellaBlock(); err == nil {
				if blindedCapellaBlk == nil {
					return nil, errNilBlock
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(blindedCapellaBlk.Block)
				if err != nil {
					return nil, errors.Wrapf(err, "could not get signed beacon block")
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
				data := &ethpbv2.SignedBlindedBeaconBlockCapella{
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
	}

	if capellaBlk == nil {
		return nil, errNilBlock
	}
	blindedBlkInterface, err := blk.ToBlinded()
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert block to blinded block")
	}
	blindedCapellaBlock, err := blindedBlkInterface.PbBlindedCapellaBlock()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(blindedCapellaBlock.Block)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
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
	data := &ethpbv2.SignedBlindedBeaconBlockCapella{
		Message:   v2Blk,
		Signature: sig[:],
	}
	sszData, err := data.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return &ethpbv2.SSZContainer{Version: ethpbv2.Version_CAPELLA, ExecutionOptimistic: isOptimistic, Data: sszData}, nil
}

func (bs *Server) getBlindedSSZBlockDeneb(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.SSZContainer, error) {
	denebBlk, err := blk.PbDenebBlock()
	if err != nil {
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			if blindedDenebBlk, err := blk.PbBlindedDenebBlock(); err == nil {
				if blindedDenebBlk == nil {
					return nil, errNilBlock
				}
				v2Blk, err := migration.V1Alpha1BeaconBlockBlindedDenebToV2Blinded(blindedDenebBlk.Block)
				if err != nil {
					return nil, errors.Wrapf(err, "could not get signed beacon block")
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
				data := &ethpbv2.SignedBlindedBeaconBlockDeneb{
					Message:   v2Blk,
					Signature: sig[:],
				}
				sszData, err := data.MarshalSSZ()
				if err != nil {
					return nil, errors.Wrapf(err, "could not marshal block into SSZ")
				}
				return &ethpbv2.SSZContainer{
					Version:             ethpbv2.Version_DENEB,
					ExecutionOptimistic: isOptimistic,
					Data:                sszData,
				}, nil
			}
			return nil, err
		}
	}

	if denebBlk == nil {
		return nil, errNilBlock
	}
	blindedBlkInterface, err := blk.ToBlinded()
	if err != nil {
		return nil, errors.Wrapf(err, "could not convert block to blinded block")
	}
	blindedDenebBlock, err := blindedBlkInterface.PbBlindedDenebBlock()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
	}
	v2Blk, err := migration.V1Alpha1BeaconBlockBlindedDenebToV2Blinded(blindedDenebBlock.Block)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get signed beacon block")
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
	data := &ethpbv2.SignedBlindedBeaconBlockDeneb{
		Message:   v2Blk,
		Signature: sig[:],
	}
	sszData, err := data.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return &ethpbv2.SSZContainer{Version: ethpbv2.Version_DENEB, ExecutionOptimistic: isOptimistic, Data: sszData}, nil
}

func (bs *Server) submitBlindedBellatrixBlock(ctx context.Context, blindedBellatrixBlk *ethpbv2.BlindedBeaconBlockBellatrix, sig []byte) error {
	b, err := migration.BlindedBellatrixToV1Alpha1SignedBlock(&ethpbv2.SignedBlindedBeaconBlockBellatrix{
		Message:   blindedBellatrixBlk,
		Signature: sig,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "Could not convert block: %v", err)
	}
	_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{
			BlindedBellatrix: b,
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Errorf(codes.Internal, "Could not propose blinded block: %v", err)
	}
	return nil
}

func (bs *Server) submitBlindedCapellaBlock(ctx context.Context, blindedCapellaBlk *ethpbv2.BlindedBeaconBlockCapella, sig []byte) error {
	b, err := migration.BlindedCapellaToV1Alpha1SignedBlock(&ethpbv2.SignedBlindedBeaconBlockCapella{
		Message:   blindedCapellaBlk,
		Signature: sig,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "Could not convert block: %v", err)
	}
	_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_BlindedCapella{
			BlindedCapella: b,
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Errorf(codes.Internal, "Could not propose blinded block: %v", err)
	}
	return nil
}

func (bs *Server) submitBlindedDenebContents(ctx context.Context, blindedDenebContents *ethpbv2.SignedBlindedBeaconBlockContentsDeneb) error {
	blk, err := migration.BlindedDenebToV1Alpha1SignedBlock(&ethpbv2.SignedBlindedBeaconBlockDeneb{
		Message:   blindedDenebContents.SignedBlindedBlock.Message,
		Signature: blindedDenebContents.SignedBlindedBlock.Signature,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	blobs := migration.SignedBlindedBlobsToV1Alpha1SignedBlindedBlobs(blindedDenebContents.SignedBlindedBlobSidecars)
	_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_BlindedDeneb{
			BlindedDeneb: &eth.SignedBlindedBeaconBlockAndBlobsDeneb{
				Block: blk,
				Blobs: blobs,
			},
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), validator.CouldNotDecodeBlock) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Errorf(codes.Internal, "Could not propose blinded block: %v", err)
	}
	return nil
}
