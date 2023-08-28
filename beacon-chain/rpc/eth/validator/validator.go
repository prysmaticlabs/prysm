package validator

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/kv"
	rpchelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var errInvalidValIndex = errors.New("invalid validator index")
var errParticipation = status.Error(codes.Internal, "Could not obtain epoch participation")

// GetSyncCommitteeDuties provides a set of sync committee duties for a particular epoch.
//
// The logic for calculating epoch validity comes from https://ethereum.github.io/beacon-APIs/?urls.primaryName=dev#/Validator/getSyncCommitteeDuties
// where `epoch` is described as `epoch // EPOCHS_PER_SYNC_COMMITTEE_PERIOD <= current_epoch // EPOCHS_PER_SYNC_COMMITTEE_PERIOD + 1`.
//
// Algorithm:
//   - Get the last valid epoch. This is the last epoch of the next sync committee period.
//   - Get the state for the requested epoch. If it's a future epoch from the current sync committee period
//     or an epoch from the next sync committee period, then get the current state.
//   - Get the state's current sync committee. If it's an epoch from the next sync committee period, then get the next sync committee.
//   - Get duties.
func (vs *Server) GetSyncCommitteeDuties(ctx context.Context, req *ethpbv2.SyncCommitteeDutiesRequest) (*ethpbv2.SyncCommitteeDutiesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.GetSyncCommitteeDuties")
	defer span.End()

	if err := rpchelpers.ValidateSyncGRPC(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	currentEpoch := slots.ToEpoch(vs.TimeFetcher.CurrentSlot())
	lastValidEpoch := syncCommitteeDutiesLastValidEpoch(currentEpoch)
	if req.Epoch > lastValidEpoch {
		return nil, status.Errorf(codes.InvalidArgument, "Epoch is too far in the future. Maximum valid epoch is %v.", lastValidEpoch)
	}

	requestedEpoch := req.Epoch
	if requestedEpoch > currentEpoch {
		requestedEpoch = currentEpoch
	}
	slot, err := slots.EpochStart(requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync committee slot: %v", err)
	}
	st, err := vs.Stater.State(ctx, []byte(strconv.FormatUint(uint64(slot), 10)))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync committee state: %v", err)
	}

	currentSyncCommitteeFirstEpoch, err := slots.SyncCommitteePeriodStartEpoch(requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not get sync committee period start epoch: %v.", err)
	}
	nextSyncCommitteeFirstEpoch := currentSyncCommitteeFirstEpoch + params.BeaconConfig().EpochsPerSyncCommitteePeriod
	var committee *ethpbalpha.SyncCommittee
	if req.Epoch >= nextSyncCommitteeFirstEpoch {
		committee, err = st.NextSyncCommittee()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get sync committee: %v", err)
		}
	} else {
		committee, err = st.CurrentSyncCommittee()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get sync committee: %v", err)
		}
	}
	committeePubkeys := make(map[[fieldparams.BLSPubkeyLength]byte][]uint64)
	for j, pubkey := range committee.Pubkeys {
		pubkey48 := bytesutil.ToBytes48(pubkey)
		committeePubkeys[pubkey48] = append(committeePubkeys[pubkey48], uint64(j))
	}

	duties, err := syncCommitteeDuties(req.Index, st, committeePubkeys)
	if errors.Is(err, errInvalidValIndex) {
		return nil, status.Error(codes.InvalidArgument, "Invalid validator index")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get duties: %v", err)
	}

	isOptimistic, err := rpchelpers.IsOptimistic(
		ctx,
		[]byte(strconv.FormatUint(uint64(slot), 10)),
		vs.OptimisticModeFetcher,
		vs.Stater,
		vs.ChainInfoFetcher,
		vs.BeaconDB,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	return &ethpbv2.SyncCommitteeDutiesResponse{
		Data:                duties,
		ExecutionOptimistic: isOptimistic,
	}, nil
}

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
	return nil, status.Error(codes.InvalidArgument, "Unsupported block type")
}

// PrepareBeaconProposer caches and updates the fee recipient for the given proposer.
func (vs *Server) PrepareBeaconProposer(
	ctx context.Context, request *ethpbv1.PrepareBeaconProposerRequest,
) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "validator.PrepareBeaconProposer")
	defer span.End()
	var feeRecipients []common.Address
	var validatorIndices []primitives.ValidatorIndex
	newRecipients := make([]*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer, 0, len(request.Recipients))
	for _, r := range request.Recipients {
		f, err := vs.BeaconDB.FeeRecipientByValidatorID(ctx, r.ValidatorIndex)
		switch {
		case errors.Is(err, kv.ErrNotFoundFeeRecipient):
			newRecipients = append(newRecipients, r)
		case err != nil:
			return nil, status.Errorf(codes.Internal, "Could not get fee recipient by validator index: %v", err)
		default:
			if common.BytesToAddress(r.FeeRecipient) != f {
				newRecipients = append(newRecipients, r)
			}
		}
	}
	if len(newRecipients) == 0 {
		return &emptypb.Empty{}, nil
	}
	for _, recipientContainer := range newRecipients {
		recipient := hexutil.Encode(recipientContainer.FeeRecipient)
		if !common.IsHexAddress(recipient) {
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid fee recipient address: %v", recipient))
		}
		feeRecipients = append(feeRecipients, common.BytesToAddress(recipientContainer.FeeRecipient))
		validatorIndices = append(validatorIndices, recipientContainer.ValidatorIndex)
	}
	if err := vs.BeaconDB.SaveFeeRecipientsByValidatorIDs(ctx, validatorIndices, feeRecipients); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not save fee recipients: %v", err)
	}

	log.WithFields(log.Fields{
		"validatorIndices": validatorIndices,
	}).Info("Updated fee recipient addresses for validator indices")
	return &emptypb.Empty{}, nil
}

// GetLiveness requests the beacon node to indicate if a validator has been observed to be live in a given epoch.
// The beacon node might detect liveness by observing messages from the validator on the network,
// in the beacon chain, from its API or from any other source.
// A beacon node SHOULD support the current and previous epoch, however it MAY support earlier epoch.
// It is important to note that the values returned by the beacon node are not canonical;
// they are best-effort and based upon a subjective view of the network.
// A beacon node that was recently started or suffered a network partition may indicate that a validator is not live when it actually is.
func (vs *Server) GetLiveness(ctx context.Context, req *ethpbv2.GetLivenessRequest) (*ethpbv2.GetLivenessResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.GetLiveness")
	defer span.End()

	var participation []byte

	// First we check if the requested epoch is the current epoch.
	// If it is, then we won't be able to fetch the state at the end of the epoch.
	// In that case we get participation info from the head state.
	// We can also use the head state to get participation info for the previous epoch.
	headSt, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	currEpoch := slots.ToEpoch(headSt.Slot())
	if req.Epoch > currEpoch {
		return nil, status.Error(codes.InvalidArgument, "Requested epoch cannot be in the future")
	}

	var st state.BeaconState
	if req.Epoch == currEpoch {
		st = headSt
		participation, err = st.CurrentEpochParticipation()
		if err != nil {
			return nil, errParticipation
		}
	} else if req.Epoch == currEpoch-1 {
		st = headSt
		participation, err = st.PreviousEpochParticipation()
		if err != nil {
			return nil, errParticipation
		}
	} else {
		epochEnd, err := slots.EpochEnd(req.Epoch)
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get requested epoch's end slot")
		}
		st, err = vs.Stater.StateBySlot(ctx, epochEnd)
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get slot for requested epoch")
		}
		participation, err = st.CurrentEpochParticipation()
		if err != nil {
			return nil, errParticipation
		}
	}

	resp := &ethpbv2.GetLivenessResponse{
		Data: make([]*ethpbv2.GetLivenessResponse_Liveness, len(req.Index)),
	}
	for i, vi := range req.Index {
		if vi >= primitives.ValidatorIndex(len(participation)) {
			return nil, status.Errorf(codes.InvalidArgument, "Validator index %d is invalid", vi)
		}
		resp.Data[i] = &ethpbv2.GetLivenessResponse_Liveness{
			Index:  vi,
			IsLive: participation[vi] != 0,
		}
	}

	return resp, nil
}

func syncCommitteeDutiesLastValidEpoch(currentEpoch primitives.Epoch) primitives.Epoch {
	currentSyncPeriodIndex := currentEpoch / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	// Return the last epoch of the next sync committee.
	// To do this we go two periods ahead to find the first invalid epoch, and then subtract 1.
	return (currentSyncPeriodIndex+2)*params.BeaconConfig().EpochsPerSyncCommitteePeriod - 1
}

func syncCommitteeDuties(
	valIndices []primitives.ValidatorIndex,
	st state.BeaconState,
	committeePubkeys map[[fieldparams.BLSPubkeyLength]byte][]uint64,
) ([]*ethpbv2.SyncCommitteeDuty, error) {
	duties := make([]*ethpbv2.SyncCommitteeDuty, 0)
	for _, index := range valIndices {
		duty := &ethpbv2.SyncCommitteeDuty{
			ValidatorIndex: index,
		}
		valPubkey48 := st.PubkeyAtIndex(index)
		var zeroPubkey [fieldparams.BLSPubkeyLength]byte
		if bytes.Equal(valPubkey48[:], zeroPubkey[:]) {
			return nil, errInvalidValIndex
		}
		valPubkey := valPubkey48[:]
		duty.Pubkey = valPubkey
		indices, ok := committeePubkeys[valPubkey48]
		if ok {
			duty.ValidatorSyncCommitteeIndices = indices
			duties = append(duties, duty)
		}
	}
	return duties, nil
}
