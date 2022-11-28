package beacon

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBlindedBlock retrieves blinded block for given block id.
func (bs *Server) GetBlindedBlock(ctx context.Context, req *ethpbv1.BlockRequest) (*ethpbv2.BlindedBlockResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetBlindedBlock")
	defer span.End()

	blk, err := bs.blockFromBlockID(ctx, req.BlockId)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, err
	}

	result, err := getBlindedBlockPhase0(blk)
	if result != nil {
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, blocks.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = getBlindedBlockAltair(blk)
	if result != nil {
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, blocks.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = bs.getBlindedBlockBellatrix(ctx, blk)
	if result != nil {
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, blocks.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}
	result, err = bs.getBlindedBlockCapella(ctx, blk)
	if result != nil {
		return result, nil
	}
	// ErrUnsupportedGetter means that we have another block type
	if !errors.Is(err, blocks.ErrUnsupportedGetter) {
		return nil, status.Errorf(codes.Internal, "Could not get blinded block: %v", err)
	}

	return nil, status.Errorf(codes.Internal, "Unknown block type %T", blk)
}

func getBlindedBlockPhase0(blk interfaces.SignedBeaconBlock) (*ethpbv2.BlindedBlockResponse, error) {
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

func getBlindedBlockAltair(blk interfaces.SignedBeaconBlock) (*ethpbv2.BlindedBlockResponse, error) {
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

func (bs *Server) getBlindedBlockBellatrix(ctx context.Context, blk interfaces.SignedBeaconBlock) (*ethpbv2.BlindedBlockResponse, error) {
	bellatrixBlk, err := blk.PbBellatrixBlock()
	if err != nil {
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, blocks.ErrUnsupportedGetter) {
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

func (bs *Server) getBlindedBlockCapella(ctx context.Context, blk interfaces.SignedBeaconBlock) (*ethpbv2.BlindedBlockResponse, error) {
	capellaBlk, err := blk.PbCapellaBlock()
	if err != nil {
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, blocks.ErrUnsupportedGetter) {
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
