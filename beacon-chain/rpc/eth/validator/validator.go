package validator

import (
	"context"
	"fmt"

	rpchelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProduceBlockV2 requests the beacon node to produce a valid unsigned beacon block, which can then be signed by a proposer and submitted.
// By definition `/eth/v2/validator/blocks/{slot}`, does not produce block using mev-boost and relayer network.
// The following endpoint states that the returned object is a BeaconBlock, not a BlindedBeaconBlock. As such, the block must return a full ExecutionPayload:
// https://ethereum.github.io/beacon-APIs/?urls.primaryName=v2.3.0#/Validator/produceBlockV2
//
// To use mev-boost and relayer network. It's recommended to use the following endpoint:
// https://github.com/ethereum/beacon-APIs/blob/master/apis/validator/blinded_block.yaml
func (vs *Server) ProduceBlockV2(ctx context.Context, req *ethpbv1.ProduceBlockRequest) (*ethpbv2.ProduceBlockResponseV2, error) {
	ctx, span := trace.StartSpan(ctx, "validator.ProduceBlockV2")
	defer span.End()

	if err := rpchelpers.ValidateSyncGRPC(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	v1alpha1req := &ethpbalpha.BlockRequest{
		Slot:         req.Slot,
		RandaoReveal: req.RandaoReveal,
		Graffiti:     req.Graffiti,
		SkipMevBoost: true, // Skip mev-boost and relayer network
	}
	v1alpha1resp, err := vs.V1Alpha1Server.GetBeaconBlock(ctx, v1alpha1req)
	if err != nil {
		// We simply return err because it's already of a gRPC error type.
		return nil, err
	}
	phase0Block, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Phase0)
	if ok {
		block, err := migration.V1Alpha1ToV1Block(phase0Block.Phase0)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		return &ethpbv2.ProduceBlockResponseV2{
			Version: ethpbv2.Version_PHASE0,
			Data: &ethpbv2.BeaconBlockContainerV2{
				Block: &ethpbv2.BeaconBlockContainerV2_Phase0Block{Phase0Block: block},
			},
		}, nil
	}
	altairBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Altair)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlock.Altair)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		return &ethpbv2.ProduceBlockResponseV2{
			Version: ethpbv2.Version_ALTAIR,
			Data: &ethpbv2.BeaconBlockContainerV2{
				Block: &ethpbv2.BeaconBlockContainerV2_AltairBlock{AltairBlock: block},
			},
		}, nil
	}
	optimistic, err := vs.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if the node is a optimistic node: %v", err)
	}
	if optimistic {
		return nil, status.Errorf(codes.Unavailable, "The node is currently optimistic and cannot serve validators")
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedBellatrix)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Bellatrix beacon block is blinded")
	}
	bellatrixBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Bellatrix)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockBellatrixToV2(bellatrixBlock.Bellatrix)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		return &ethpbv2.ProduceBlockResponseV2{
			Version: ethpbv2.Version_BELLATRIX,
			Data: &ethpbv2.BeaconBlockContainerV2{
				Block: &ethpbv2.BeaconBlockContainerV2_BellatrixBlock{BellatrixBlock: block},
			},
		}, nil
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedCapella)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Capella beacon block is blinded")
	}
	capellaBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Capella)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockCapellaToV2(capellaBlock.Capella)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		return &ethpbv2.ProduceBlockResponseV2{
			Version: ethpbv2.Version_CAPELLA,
			Data: &ethpbv2.BeaconBlockContainerV2{
				Block: &ethpbv2.BeaconBlockContainerV2_CapellaBlock{CapellaBlock: block},
			},
		}, nil
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedDeneb)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Deneb beacon block contents are blinded")
	}
	denebBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Deneb)
	if ok {
		blockAndBlobs, err := migration.V1Alpha1BeaconBlockDenebAndBlobsToV2(denebBlock.Deneb)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block contents: %v", err)
		}
		return &ethpbv2.ProduceBlockResponseV2{
			Version: ethpbv2.Version_DENEB,
			Data: &ethpbv2.BeaconBlockContainerV2{
				Block: &ethpbv2.BeaconBlockContainerV2_DenebContents{
					DenebContents: &ethpbv2.BeaconBlockContentsDeneb{
						Block:        blockAndBlobs.Block,
						BlobSidecars: blockAndBlobs.BlobSidecars,
					}},
			},
		}, nil
	}
	return nil, status.Error(codes.InvalidArgument, "Unsupported block type")
}

// ProduceBlockV2SSZ requests the beacon node to produce a valid unsigned beacon block, which can then be signed by a proposer and submitted.
//
// The produced block is in SSZ form.
// By definition `/eth/v2/validator/blocks/{slot}/ssz`, does not produce block using mev-boost and relayer network:
// The following endpoint states that the returned object is a BeaconBlock, not a BlindedBeaconBlock. As such, the block must return a full ExecutionPayload:
// https://ethereum.github.io/beacon-APIs/?urls.primaryName=v2.3.0#/Validator/produceBlockV2
//
// To use mev-boost and relayer network. It's recommended to use the following endpoint:
// https://github.com/ethereum/beacon-APIs/blob/master/apis/validator/blinded_block.yaml
func (vs *Server) ProduceBlockV2SSZ(ctx context.Context, req *ethpbv1.ProduceBlockRequest) (*ethpbv2.SSZContainer, error) {
	ctx, span := trace.StartSpan(ctx, "validator.ProduceBlockV2SSZ")
	defer span.End()

	if err := rpchelpers.ValidateSyncGRPC(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	v1alpha1req := &ethpbalpha.BlockRequest{
		Slot:         req.Slot,
		RandaoReveal: req.RandaoReveal,
		Graffiti:     req.Graffiti,
		SkipMevBoost: true, // Skip mev-boost and relayer network
	}
	v1alpha1resp, err := vs.V1Alpha1Server.GetBeaconBlock(ctx, v1alpha1req)
	if err != nil {
		// We simply return err because it's already of a gRPC error type.
		return nil, err
	}
	phase0Block, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Phase0)
	if ok {
		block, err := migration.V1Alpha1ToV1Block(phase0Block.Phase0)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := block.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_PHASE0,
			Data:    sszBlock,
		}, nil
	}
	altairBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Altair)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlock.Altair)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := block.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_ALTAIR,
			Data:    sszBlock,
		}, nil
	}
	optimistic, err := vs.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if the node is a optimistic node: %v", err)
	}
	if optimistic {
		return nil, status.Errorf(codes.Unavailable, "The node is currently optimistic and cannot serve validators")
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedBellatrix)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Bellatrix beacon block is blinded")
	}
	bellatrixBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Bellatrix)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockBellatrixToV2(bellatrixBlock.Bellatrix)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := block.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_BELLATRIX,
			Data:    sszBlock,
		}, nil
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedCapella)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Capella beacon block is blinded")
	}
	capellaBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Capella)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockCapellaToV2(capellaBlock.Capella)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := block.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_CAPELLA,
			Data:    sszBlock,
		}, nil
	}

	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedDeneb)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Deneb beacon blockcontent is blinded")
	}
	denebBlockcontent, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Deneb)
	if ok {
		blockContent, err := migration.V1Alpha1BeaconBlockDenebAndBlobsToV2(denebBlockcontent.Deneb)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := blockContent.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_DENEB,
			Data:    sszBlock,
		}, nil
	}

	return nil, status.Error(codes.InvalidArgument, "Unsupported block type")
}

// ProduceBlindedBlock requests the beacon node to produce a valid unsigned blinded beacon block,
// which can then be signed by a proposer and submitted.
//
// Under the following conditions, this endpoint will return an error.
// - The node is syncing or optimistic mode (after bellatrix).
// - The builder is not figured (after bellatrix).
// - The relayer circuit breaker is activated (after bellatrix).
// - The relayer responded with an error (after bellatrix).
func (vs *Server) ProduceBlindedBlock(ctx context.Context, req *ethpbv1.ProduceBlockRequest) (*ethpbv2.ProduceBlindedBlockResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.ProduceBlindedBlock")
	defer span.End()

	if !vs.BlockBuilder.Configured() {
		return nil, status.Error(codes.Internal, "Block builder not configured")
	}
	if err := rpchelpers.ValidateSyncGRPC(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	v1alpha1req := &ethpbalpha.BlockRequest{
		Slot:         req.Slot,
		RandaoReveal: req.RandaoReveal,
		Graffiti:     req.Graffiti,
	}
	v1alpha1resp, err := vs.V1Alpha1Server.GetBeaconBlock(ctx, v1alpha1req)
	if err != nil {
		// We simply return err because it's already of a gRPC error type.
		return nil, err
	}

	phase0Block, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Phase0)
	if ok {
		block, err := migration.V1Alpha1ToV1Block(phase0Block.Phase0)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		return &ethpbv2.ProduceBlindedBlockResponse{
			Version: ethpbv2.Version_PHASE0,
			Data: &ethpbv2.BlindedBeaconBlockContainer{
				Block: &ethpbv2.BlindedBeaconBlockContainer_Phase0Block{Phase0Block: block},
			},
		}, nil
	}
	altairBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Altair)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlock.Altair)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		return &ethpbv2.ProduceBlindedBlockResponse{
			Version: ethpbv2.Version_ALTAIR,
			Data: &ethpbv2.BlindedBeaconBlockContainer{
				Block: &ethpbv2.BlindedBeaconBlockContainer_AltairBlock{AltairBlock: block},
			},
		}, nil
	}
	optimistic, err := vs.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if the node is a optimistic node: %v", err)
	}
	if optimistic {
		return nil, status.Errorf(codes.Unavailable, "The node is currently optimistic and cannot serve validators")
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Bellatrix)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared beacon block is not blinded")
	}
	bellatrixBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedBellatrix)
	if ok {
		blk, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(bellatrixBlock.BlindedBellatrix)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		return &ethpbv2.ProduceBlindedBlockResponse{
			Version: ethpbv2.Version_BELLATRIX,
			Data: &ethpbv2.BlindedBeaconBlockContainer{
				Block: &ethpbv2.BlindedBeaconBlockContainer_BellatrixBlock{BellatrixBlock: blk},
			},
		}, nil
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Capella)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared beacon block is not blinded")
	}
	capellaBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedCapella)
	if ok {
		blk, err := migration.V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(capellaBlock.BlindedCapella)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		return &ethpbv2.ProduceBlindedBlockResponse{
			Version: ethpbv2.Version_CAPELLA,
			Data: &ethpbv2.BlindedBeaconBlockContainer{
				Block: &ethpbv2.BlindedBeaconBlockContainer_CapellaBlock{CapellaBlock: blk},
			},
		}, nil
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Deneb)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Deneb beacon block contents are not blinded")
	}
	denebBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedDeneb)
	if ok {
		blockAndBlobs, err := migration.V1Alpha1BlindedBlockAndBlobsDenebToV2Blinded(denebBlock.BlindedDeneb)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block contents: %v", err)
		}
		return &ethpbv2.ProduceBlindedBlockResponse{
			Version: ethpbv2.Version_DENEB,
			Data: &ethpbv2.BlindedBeaconBlockContainer{
				Block: &ethpbv2.BlindedBeaconBlockContainer_DenebContents{
					DenebContents: &ethpbv2.BlindedBeaconBlockContentsDeneb{
						BlindedBlock:        blockAndBlobs.BlindedBlock,
						BlindedBlobSidecars: blockAndBlobs.BlindedBlobSidecars,
					}},
			},
		}, nil
	}
	return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("block was not a supported blinded block type, validator may not be registered if using a relay. received: %T", v1alpha1resp.Block))
}

// ProduceBlindedBlockSSZ requests the beacon node to produce a valid unsigned blinded beacon block,
// which can then be signed by a proposer and submitted.
//
// The produced block is in SSZ form.
//
// Pre-Bellatrix, this endpoint will return a regular block.
func (vs *Server) ProduceBlindedBlockSSZ(ctx context.Context, req *ethpbv1.ProduceBlockRequest) (*ethpbv2.SSZContainer, error) {
	ctx, span := trace.StartSpan(ctx, "validator.ProduceBlindedBlockSSZ")
	defer span.End()

	if !vs.BlockBuilder.Configured() {
		return nil, status.Error(codes.Internal, "Block builder not configured")
	}
	if err := rpchelpers.ValidateSyncGRPC(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	v1alpha1req := &ethpbalpha.BlockRequest{
		Slot:         req.Slot,
		RandaoReveal: req.RandaoReveal,
		Graffiti:     req.Graffiti,
	}
	v1alpha1resp, err := vs.V1Alpha1Server.GetBeaconBlock(ctx, v1alpha1req)
	if err != nil {
		// We simply return err because it's already of a gRPC error type.
		return nil, err
	}

	phase0Block, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Phase0)
	if ok {
		block, err := migration.V1Alpha1ToV1Block(phase0Block.Phase0)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := block.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_PHASE0,
			Data:    sszBlock,
		}, nil
	}
	altairBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Altair)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockAltairToV2(altairBlock.Altair)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := block.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_ALTAIR,
			Data:    sszBlock,
		}, nil
	}
	optimistic, err := vs.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if the node is a optimistic node: %v", err)
	}
	if optimistic {
		return nil, status.Errorf(codes.Unavailable, "The node is currently optimistic and cannot serve validators")
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Bellatrix)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Bellatrix beacon block is not blinded")
	}
	bellatrixBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedBellatrix)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(bellatrixBlock.BlindedBellatrix)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := block.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_BELLATRIX,
			Data:    sszBlock,
		}, nil
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Capella)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Capella beacon block is not blinded")
	}
	capellaBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedCapella)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockBlindedCapellaToV2Blinded(capellaBlock.BlindedCapella)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := block.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_CAPELLA,
			Data:    sszBlock,
		}, nil
	}
	_, ok = v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Deneb)
	if ok {
		return nil, status.Error(codes.Internal, "Prepared Deneb beacon block content is not blinded")
	}
	denebBlockcontent, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_BlindedDeneb)
	if ok {
		blockContent, err := migration.V1Alpha1BlindedBlockAndBlobsDenebToV2Blinded(denebBlockcontent.BlindedDeneb)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
		}
		sszBlock, err := blockContent.MarshalSSZ()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not marshal block into SSZ format: %v", err)
		}
		return &ethpbv2.SSZContainer{
			Version: ethpbv2.Version_DENEB,
			Data:    sszBlock,
		}, nil
	}
	return nil, status.Error(codes.InvalidArgument, "Unsupported block type")
}
