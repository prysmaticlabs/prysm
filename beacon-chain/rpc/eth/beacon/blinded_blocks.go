package beacon

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	consensus_types "github.com/prysmaticlabs/prysm/v4/consensus-types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz/detect"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
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

	result, err := getBlindedBlockPhase0(blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = getBlindedBlockAltair(blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = bs.getBlindedBlockBellatrix(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = bs.getBlindedBlockCapella(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
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
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = getSSZBlockAltair(blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = bs.getBlindedSSZBlockBellatrix(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get signed beacon block: %v", err)
	}
	result, err = bs.getBlindedSSZBlockCapella(ctx, blk)
	if result != nil {
		result.Finalized = bs.FinalizationFetcher.IsFinalized(ctx, blkRoot)
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
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
func (bs *Server) SubmitBlindedBlock(ctx context.Context, req *ethpbv2.SignedBlindedBeaconBlockContainer) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitBlindedBlock")
	defer span.End()

	switch blkContainer := req.Message.(type) {
	case *ethpbv2.SignedBlindedBeaconBlockContainer_CapellaBlock:
		if err := bs.submitBlindedCapellaBlock(ctx, blkContainer.CapellaBlock, req.Signature); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBlindedBeaconBlockContainer_BellatrixBlock:
		if err := bs.submitBlindedBellatrixBlock(ctx, blkContainer.BellatrixBlock, req.Signature); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBlindedBeaconBlockContainer_Phase0Block:
		if err := bs.submitPhase0Block(ctx, blkContainer.Phase0Block, req.Signature); err != nil {
			return nil, err
		}
	case *ethpbv2.SignedBlindedBeaconBlockContainer_AltairBlock:
		if err := bs.submitAltairBlock(ctx, blkContainer.AltairBlock, req.Signature); err != nil {
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

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not read"+versionHeader+" header")
	}
	ver := md.Get(versionHeader)
	if len(ver) == 0 {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not read"+versionHeader+" header")
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

	if block.IsBlinded() {
		b, err := block.PbBlindedBellatrixBlock()
		if err != nil {
			b, err := block.PbBlindedCapellaBlock()
			if err != nil {
				return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
			}
			bb, err := migration.V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(b.Block)
			if err != nil {
				return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not migrate block: %v", err)
			}
			return &emptypb.Empty{}, bs.submitBlindedCapellaBlock(ctx, bb, b.Signature)
		}
		bb, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(b.Block)
		if err != nil {
			return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not migrate block: %v", err)
		}
		return &emptypb.Empty{}, bs.submitBlindedBellatrixBlock(ctx, bb, b.Signature)
	}

	root, err := block.Block().HashTreeRoot()
	if err != nil {
		return &emptypb.Empty{}, status.Errorf(codes.Internal, "Could not compute block's hash tree root: %v", err)
	}
	return &emptypb.Empty{}, bs.submitBlock(ctx, root, block)
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
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedGetter) {
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
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedGetter) {
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

func (bs *Server) getBlindedSSZBlockBellatrix(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*ethpbv2.SSZContainer, error) {
	bellatrixBlk, err := blk.PbBellatrixBlock()
	if err != nil {
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedGetter) {
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
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedGetter) {
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

func (bs *Server) submitBlindedBellatrixBlock(ctx context.Context, blindedBellatrixBlk *ethpbv2.BlindedBeaconBlockBellatrix, sig []byte) error {
	b, err := migration.BlindedBellatrixToV1Alpha1SignedBlock(&ethpbv2.SignedBlindedBeaconBlockBellatrix{
		Message:   blindedBellatrixBlk,
		Signature: sig,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{
			BlindedBellatrix: b,
		},
	})
	if err != nil {
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
		return status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	_, err = bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, &eth.GenericSignedBeaconBlock{
		Block: &eth.GenericSignedBeaconBlock_BlindedCapella{
			BlindedCapella: b,
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "Could not propose blinded block: %v", err)
	}
	return nil
}
