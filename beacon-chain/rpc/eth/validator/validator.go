package validator

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/builder"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/kv"
	rpchelpers "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	statev1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var errInvalidValIndex = errors.New("invalid validator index")

// GetAttesterDuties requests the beacon node to provide a set of attestation duties,
// which should be performed by validators, for a particular epoch.
func (vs *Server) GetAttesterDuties(ctx context.Context, req *ethpbv1.AttesterDutiesRequest) (*ethpbv1.AttesterDutiesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.GetAttesterDuties")
	defer span.End()

	if err := rpchelpers.ValidateSync(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	cs := vs.TimeFetcher.CurrentSlot()
	currentEpoch := slots.ToEpoch(cs)
	if req.Epoch > currentEpoch+1 {
		return nil, status.Errorf(codes.InvalidArgument, "Request epoch %d can not be greater than next epoch %d", req.Epoch, currentEpoch+1)
	}

	isOptimistic, err := vs.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check optimistic status: %v", err)
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	s, err = advanceState(ctx, s, req.Epoch, currentEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not advance state to requested epoch start slot: %v", err)
	}

	committeeAssignments, _, err := helpers.CommitteeAssignments(ctx, s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}
	activeValidatorCount, err := helpers.ActiveValidatorCount(ctx, s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
	}
	committeesAtSlot := helpers.SlotCommitteeCount(activeValidatorCount)

	duties := make([]*ethpbv1.AttesterDuty, 0, len(req.Index))
	for _, index := range req.Index {
		pubkey := s.PubkeyAtIndex(index)
		zeroPubkey := [fieldparams.BLSPubkeyLength]byte{}
		if bytes.Equal(pubkey[:], zeroPubkey[:]) {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid validator index")
		}
		committee := committeeAssignments[index]
		if committee == nil {
			continue
		}
		var valIndexInCommittee types.CommitteeIndex
		// valIndexInCommittee will be 0 in case we don't get a match. This is a potential false positive,
		// however it's an impossible condition because every validator must be assigned to a committee.
		for cIndex, vIndex := range committee.Committee {
			if vIndex == index {
				valIndexInCommittee = types.CommitteeIndex(uint64(cIndex))
				break
			}
		}
		duties = append(duties, &ethpbv1.AttesterDuty{
			Pubkey:                  pubkey[:],
			ValidatorIndex:          index,
			CommitteeIndex:          committee.CommitteeIndex,
			CommitteeLength:         uint64(len(committee.Committee)),
			CommitteesAtSlot:        committeesAtSlot,
			ValidatorCommitteeIndex: valIndexInCommittee,
			Slot:                    committee.AttesterSlot,
		})
	}

	root, err := attestationDependentRoot(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get dependent root: %v", err)
	}

	return &ethpbv1.AttesterDutiesResponse{
		DependentRoot:       root,
		Data:                duties,
		ExecutionOptimistic: isOptimistic,
	}, nil
}

// GetProposerDuties requests beacon node to provide all validators that are scheduled to propose a block in the given epoch.
func (vs *Server) GetProposerDuties(ctx context.Context, req *ethpbv1.ProposerDutiesRequest) (*ethpbv1.ProposerDutiesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.GetProposerDuties")
	defer span.End()

	if err := rpchelpers.ValidateSync(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	cs := vs.TimeFetcher.CurrentSlot()
	currentEpoch := slots.ToEpoch(cs)
	if req.Epoch > currentEpoch+1 {
		return nil, status.Errorf(codes.InvalidArgument, "Request epoch %d can not be greater than next epoch %d", req.Epoch, currentEpoch+1)
	}

	isOptimistic, err := vs.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check optimistic status: %v", err)
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	s, err = advanceState(ctx, s, req.Epoch, currentEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not advance state to requested epoch start slot: %v", err)
	}

	_, proposals, err := helpers.CommitteeAssignments(ctx, s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}

	duties := make([]*ethpbv1.ProposerDuty, 0)
	for index, ss := range proposals {
		val, err := s.ValidatorAtIndexReadOnly(index)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator: %v", err)
		}
		pubkey48 := val.PublicKey()
		pubkey := pubkey48[:]
		for _, s := range ss {
			duties = append(duties, &ethpbv1.ProposerDuty{
				Pubkey:         pubkey,
				ValidatorIndex: index,
				Slot:           s,
			})
		}
	}
	sort.Slice(duties, func(i, j int) bool {
		return duties[i].Slot < duties[j].Slot
	})

	root, err := vs.proposalDependentRoot(ctx, s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get dependent root: %v", err)
	}

	return &ethpbv1.ProposerDutiesResponse{
		DependentRoot:       root,
		Data:                duties,
		ExecutionOptimistic: isOptimistic,
	}, nil
}

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

	if err := rpchelpers.ValidateSync(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
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
	st, err := vs.StateFetcher.State(ctx, []byte(strconv.FormatUint(uint64(slot), 10)))
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

	isOptimistic, err := rpchelpers.IsOptimistic(ctx, st, vs.OptimisticModeFetcher)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	return &ethpbv2.SyncCommitteeDutiesResponse{
		Data:                duties,
		ExecutionOptimistic: isOptimistic,
	}, nil
}

// ProduceBlockV2 requests the beacon node to produce a valid unsigned beacon block, which can then be signed by a proposer and submitted.
func (vs *Server) ProduceBlockV2(ctx context.Context, req *ethpbv1.ProduceBlockRequest) (*ethpbv2.ProduceBlockResponseV2, error) {
	ctx, span := trace.StartSpan(ctx, "validator.ProduceBlockV2")
	defer span.End()

	if err := rpchelpers.ValidateSync(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
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
	return nil, status.Error(codes.InvalidArgument, "Unsupported block type")
}

// ProduceBlockV2SSZ requests the beacon node to produce a valid unsigned beacon block, which can then be signed by a proposer and submitted.
//
// The produced block is in SSZ form.
func (vs *Server) ProduceBlockV2SSZ(ctx context.Context, req *ethpbv1.ProduceBlockRequest) (*ethpbv2.SSZContainer, error) {
	ctx, span := trace.StartSpan(ctx, "validator.ProduceBlockV2SSZ")
	defer span.End()

	if err := rpchelpers.ValidateSync(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
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

	if err := rpchelpers.ValidateSync(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}
	v1alpha1req := &ethpbalpha.BlockRequest{
		Slot:         req.Slot,
		RandaoReveal: req.RandaoReveal,
		Graffiti:     req.Graffiti,
	}

	// Before Bellatrix, return normal block.
	if req.Slot < types.Slot(params.BeaconConfig().BellatrixForkEpoch)*params.BeaconConfig().SlotsPerEpoch {
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
	}

	// After Bellatrix, return blinded block.
	optimistic, err := vs.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not determine if the node is a optimistic node: %v", err)
	}
	if optimistic {
		return nil, status.Errorf(codes.Unavailable, "The node is currently optimistic and cannot serve validators")
	}
	altairBlk, err := vs.V1Alpha1Server.BuildAltairBeaconBlock(ctx, v1alpha1req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not prepare beacon block: %v", err)
	}
	ok, b, err := vs.V1Alpha1Server.GetAndBuildBlindBlock(ctx, altairBlk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not prepare blind beacon block: %v", err)
	}
	if !ok {
		return nil, status.Error(codes.Unavailable, "Builder is not available due to miss-config or circuit breaker")
	}
	blk, err := migration.V1Alpha1BeaconBlockBlindedBellatrixToV2Blinded(b.GetBlindedBellatrix())
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

// ProduceBlindedBlockSSZ requests the beacon node to produce a valid unsigned blinded beacon block,
// which can then be signed by a proposer and submitted.
//
// The produced block is in SSZ form.
//
// Pre-Bellatrix, this endpoint will return a regular block.
func (vs *Server) ProduceBlindedBlockSSZ(ctx context.Context, req *ethpbv1.ProduceBlockRequest) (*ethpbv2.SSZContainer, error) {
	ctx, span := trace.StartSpan(ctx, "validator.ProduceBlindedBlockSSZ")
	defer span.End()

	if err := rpchelpers.ValidateSync(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
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
	bellatrixBlock, ok := v1alpha1resp.Block.(*ethpbalpha.GenericBeaconBlock_Bellatrix)
	if ok {
		block, err := migration.V1Alpha1BeaconBlockBellatrixToV2Blinded(bellatrixBlock.Bellatrix)
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
	return nil, status.Error(codes.InvalidArgument, "Unsupported block type")
}

// PrepareBeaconProposer caches and updates the fee recipient for the given proposer.
func (vs *Server) PrepareBeaconProposer(
	ctx context.Context, request *ethpbv1.PrepareBeaconProposerRequest,
) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "validator.PrepareBeaconProposer")
	defer span.End()
	var feeRecipients []common.Address
	var validatorIndices []types.ValidatorIndex
	newRecipients := make([]*ethpbv1.PrepareBeaconProposerRequest_FeeRecipientContainer, 0, len(request.Recipients))
	for _, r := range request.Recipients {
		f, err := vs.V1Alpha1Server.BeaconDB.FeeRecipientByValidatorID(ctx, r.ValidatorIndex)
		switch {
		case errors.Is(err, kv.ErrNotFoundFeeRecipient):
			newRecipients = append(newRecipients, r)
		case err != nil:
			return nil, status.Errorf(codes.Internal, "Could not get fee recipient by validator index: %v", err)
		default:
		}
		if common.BytesToAddress(r.FeeRecipient) != f {
			newRecipients = append(newRecipients, r)
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
	if err := vs.V1Alpha1Server.BeaconDB.SaveFeeRecipientsByValidatorIDs(ctx, validatorIndices, feeRecipients); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not save fee recipients: %v", err)
	}
	log.WithFields(log.Fields{
		"validatorIndices": validatorIndices,
	}).Info("Updated fee recipient addresses for validator indices")
	return &emptypb.Empty{}, nil
}

// SubmitValidatorRegistration submits validator registrations.
func (vs *Server) SubmitValidatorRegistration(ctx context.Context, reg *ethpbv1.SubmitValidatorRegistrationsRequest) (*empty.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitValidatorRegistration")
	defer span.End()

	if vs.V1Alpha1Server.BlockBuilder == nil || !vs.V1Alpha1Server.BlockBuilder.Configured() {
		return &empty.Empty{}, status.Errorf(codes.Internal, "Could not register block builder: %v", builder.ErrNoBuilder)
	}
	var registrations []*ethpbalpha.SignedValidatorRegistrationV1
	for i, registration := range reg.Registrations {
		message := reg.Registrations[i].Message
		registrations = append(registrations, &ethpbalpha.SignedValidatorRegistrationV1{
			Message: &ethpbalpha.ValidatorRegistrationV1{
				FeeRecipient: message.FeeRecipient,
				GasLimit:     message.GasLimit,
				Timestamp:    message.Timestamp,
				Pubkey:       message.Pubkey,
			},
			Signature: registration.Signature,
		})
	}
	if len(registrations) == 0 {
		return &empty.Empty{}, status.Errorf(codes.InvalidArgument, "Validator registration request is empty")
	}

	if err := vs.V1Alpha1Server.BlockBuilder.RegisterValidator(ctx, registrations); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not register block builder: %v", err)
	}

	return &empty.Empty{}, nil
}

// ProduceAttestationData requests that the beacon node produces attestation data for
// the requested committee index and slot based on the nodes current head.
func (vs *Server) ProduceAttestationData(ctx context.Context, req *ethpbv1.ProduceAttestationDataRequest) (*ethpbv1.ProduceAttestationDataResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.ProduceAttestationData")
	defer span.End()

	v1alpha1req := &ethpbalpha.AttestationDataRequest{
		Slot:           req.Slot,
		CommitteeIndex: req.CommitteeIndex,
	}
	v1alpha1resp, err := vs.V1Alpha1Server.GetAttestationData(ctx, v1alpha1req)
	if err != nil {
		// We simply return err because it's already of a gRPC error type.
		return nil, err
	}
	attData := migration.V1Alpha1AttDataToV1(v1alpha1resp)

	return &ethpbv1.ProduceAttestationDataResponse{Data: attData}, nil
}

// GetAggregateAttestation aggregates all attestations matching the given attestation data root and slot, returning the aggregated result.
func (vs *Server) GetAggregateAttestation(ctx context.Context, req *ethpbv1.AggregateAttestationRequest) (*ethpbv1.AggregateAttestationResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.GetAggregateAttestation")
	defer span.End()

	allAtts := vs.AttestationsPool.AggregatedAttestations()
	var bestMatchingAtt *ethpbalpha.Attestation
	for _, att := range allAtts {
		if att.Data.Slot == req.Slot {
			root, err := att.Data.HashTreeRoot()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get attestation data root: %v", err)
			}
			if bytes.Equal(root[:], req.AttestationDataRoot) {
				if bestMatchingAtt == nil || len(att.AggregationBits) > len(bestMatchingAtt.AggregationBits) {
					bestMatchingAtt = att
				}
			}
		}
	}

	if bestMatchingAtt == nil {
		return nil, status.Error(codes.NotFound, "No matching attestation found")
	}
	return &ethpbv1.AggregateAttestationResponse{
		Data: migration.V1Alpha1AttestationToV1(bestMatchingAtt),
	}, nil
}

// SubmitAggregateAndProofs verifies given aggregate and proofs and publishes them on appropriate gossipsub topic.
func (vs *Server) SubmitAggregateAndProofs(ctx context.Context, req *ethpbv1.SubmitAggregateAndProofsRequest) (*empty.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitAggregateAndProofs")
	defer span.End()

	for _, agg := range req.Data {
		if agg == nil || agg.Message == nil || agg.Message.Aggregate == nil || agg.Message.Aggregate.Data == nil {
			return nil, status.Error(codes.InvalidArgument, "Signed aggregate request can't be nil")
		}
		sigLen := fieldparams.BLSSignatureLength
		emptySig := make([]byte, sigLen)
		if bytes.Equal(agg.Signature, emptySig) || bytes.Equal(agg.Message.SelectionProof, emptySig) || bytes.Equal(agg.Message.Aggregate.Signature, emptySig) {
			return nil, status.Error(codes.InvalidArgument, "Signed signatures can't be zero hashes")
		}
		if len(agg.Signature) != sigLen || len(agg.Message.Aggregate.Signature) != sigLen {
			return nil, status.Errorf(codes.InvalidArgument, "Incorrect signature length. Expected %d bytes", sigLen)
		}

		// As a preventive measure, a beacon node shouldn't broadcast an attestation whose slot is out of range.
		if err := helpers.ValidateAttestationTime(agg.Message.Aggregate.Data.Slot,
			vs.TimeFetcher.GenesisTime(), params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
			return nil, status.Error(codes.InvalidArgument, "Attestation slot is no longer valid from current time")
		}
	}

	broadcastFailed := false
	for _, agg := range req.Data {
		v1alpha1Agg := migration.V1SignedAggregateAttAndProofToV1Alpha1(agg)
		if err := vs.Broadcaster.Broadcast(ctx, v1alpha1Agg); err != nil {
			broadcastFailed = true
		} else {
			log.WithFields(log.Fields{
				"slot":            agg.Message.Aggregate.Data.Slot,
				"committeeIndex":  agg.Message.Aggregate.Data.Index,
				"validatorIndex":  agg.Message.AggregatorIndex,
				"aggregatedCount": agg.Message.Aggregate.AggregationBits.Count(),
			}).Debug("Broadcasting aggregated attestation and proof")
		}
	}

	if broadcastFailed {
		return nil, status.Errorf(
			codes.Internal,
			"Could not broadcast one or more signed aggregated attestations")
	}

	return &emptypb.Empty{}, nil
}

// SubmitBeaconCommitteeSubscription searches using discv5 for peers related to the provided subnet information
// and replaces current peers with those ones if necessary.
func (vs *Server) SubmitBeaconCommitteeSubscription(ctx context.Context, req *ethpbv1.SubmitBeaconCommitteeSubscriptionsRequest) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitBeaconCommitteeSubscription")
	defer span.End()

	if err := rpchelpers.ValidateSync(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	if len(req.Data) == 0 {
		return nil, status.Error(codes.InvalidArgument, "No subscriptions provided")
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Verify validators at the beginning to return early if request is invalid.
	validators := make([]state.ReadOnlyValidator, len(req.Data))
	for i, sub := range req.Data {
		val, err := s.ValidatorAtIndexReadOnly(sub.ValidatorIndex)
		if outOfRangeErr, ok := err.(*statev1.ValidatorIndexOutOfRangeError); ok {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid validator ID: %v", outOfRangeErr)
		}
		validators[i] = val
	}

	fetchValsLen := func(slot types.Slot) (uint64, error) {
		wantedEpoch := slots.ToEpoch(slot)
		vals, err := vs.HeadFetcher.HeadValidatorsIndices(ctx, wantedEpoch)
		if err != nil {
			return 0, err
		}
		return uint64(len(vals)), nil
	}

	// Request the head validator indices of epoch represented by the first requested slot.
	currValsLen, err := fetchValsLen(req.Data[0].Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head validator length: %v", err)
	}
	currEpoch := slots.ToEpoch(req.Data[0].Slot)

	for _, sub := range req.Data {
		// If epoch has changed, re-request active validators length
		if currEpoch != slots.ToEpoch(sub.Slot) {
			currValsLen, err = fetchValsLen(sub.Slot)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve head validator length: %v", err)
			}
			currEpoch = slots.ToEpoch(sub.Slot)
		}
		subnet := helpers.ComputeSubnetFromCommitteeAndSlot(currValsLen, sub.CommitteeIndex, sub.Slot)
		cache.SubnetIDs.AddAttesterSubnetID(sub.Slot, subnet)
		if sub.IsAggregator {
			cache.SubnetIDs.AddAggregatorSubnetID(sub.Slot, subnet)
		}
	}

	for _, val := range validators {
		valStatus, err := rpchelpers.ValidatorStatus(val, currEpoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve validator status: %v", err)
		}
		pubkey := val.PublicKey()
		vs.V1Alpha1Server.AssignValidatorToSubnet(pubkey[:], v1ValidatorStatusToV1Alpha1(valStatus))
	}

	return &emptypb.Empty{}, nil
}

// SubmitSyncCommitteeSubscription subscribe to a number of sync committee subnets.
//
// Subscribing to sync committee subnets is an action performed by VC to enable
// network participation in Altair networks, and only required if the VC has an active
// validator in an active sync committee.
func (vs *Server) SubmitSyncCommitteeSubscription(ctx context.Context, req *ethpbv2.SubmitSyncCommitteeSubscriptionsRequest) (*empty.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitSyncCommitteeSubscription")
	defer span.End()

	if err := rpchelpers.ValidateSync(ctx, vs.SyncChecker, vs.HeadFetcher, vs.TimeFetcher, vs.OptimisticModeFetcher); err != nil {
		// We simply return the error because it's already a gRPC error.
		return nil, err
	}

	if len(req.Data) == 0 {
		return nil, status.Error(codes.InvalidArgument, "No subscriptions provided")
	}
	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	currEpoch := slots.ToEpoch(s.Slot())
	validators := make([]state.ReadOnlyValidator, len(req.Data))
	for i, sub := range req.Data {
		val, err := s.ValidatorAtIndexReadOnly(sub.ValidatorIndex)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator at index %d: %v", sub.ValidatorIndex, err)
		}
		valStatus, err := rpchelpers.ValidatorSubStatus(val, currEpoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get validator status at index %d: %v", sub.ValidatorIndex, err)
		}
		if valStatus != ethpbv1.ValidatorStatus_ACTIVE_ONGOING && valStatus != ethpbv1.ValidatorStatus_ACTIVE_EXITING {
			return nil, status.Errorf(codes.InvalidArgument, "Validator at index %d is not active or exiting: %v", sub.ValidatorIndex, err)
		}
		validators[i] = val
	}

	startEpoch, err := slots.SyncCommitteePeriodStartEpoch(currEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync committee period start epoch: %v.", err)
	}

	for i, sub := range req.Data {
		if sub.UntilEpoch <= currEpoch {
			return nil, status.Errorf(codes.InvalidArgument, "Epoch for subscription at index %d is in the past. It must be at least %d", i, currEpoch+1)
		}
		maxValidUntilEpoch := startEpoch + params.BeaconConfig().EpochsPerSyncCommitteePeriod*2
		if sub.UntilEpoch > maxValidUntilEpoch {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"Epoch for subscription at index %d is too far in the future. It can be at most %d",
				i,
				maxValidUntilEpoch,
			)
		}
	}

	for i, sub := range req.Data {
		pubkey48 := validators[i].PublicKey()
		// Handle overflow in the event current epoch is less than end epoch.
		// This is an impossible condition, so it is a defensive check.
		epochsToWatch, err := sub.UntilEpoch.SafeSub(uint64(startEpoch))
		if err != nil {
			epochsToWatch = 0
		}
		epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot)) * time.Second
		totalDuration := epochDuration * time.Duration(epochsToWatch)

		cache.SyncSubnetIDs.AddSyncCommitteeSubnets(pubkey48[:], startEpoch, sub.SyncCommitteeIndices, totalDuration)
	}

	return &empty.Empty{}, nil
}

// ProduceSyncCommitteeContribution requests that the beacon node produce a sync committee contribution.
func (vs *Server) ProduceSyncCommitteeContribution(
	ctx context.Context,
	req *ethpbv2.ProduceSyncCommitteeContributionRequest,
) (*ethpbv2.ProduceSyncCommitteeContributionResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.ProduceSyncCommitteeContribution")
	defer span.End()

	msgs, err := vs.SyncCommitteePool.SyncCommitteeMessages(req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync subcommittee messages: %v", err)
	}
	if msgs == nil {
		return nil, status.Errorf(codes.NotFound, "No subcommittee messages found")
	}
	aggregatedSig, bits, err := vs.V1Alpha1Server.AggregatedSigAndAggregationBits(ctx, msgs, req.Slot, req.SubcommitteeIndex, req.BeaconBlockRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get contribution data: %v", err)
	}
	contribution := &ethpbv2.SyncCommitteeContribution{
		Slot:              req.Slot,
		BeaconBlockRoot:   req.BeaconBlockRoot,
		SubcommitteeIndex: req.SubcommitteeIndex,
		AggregationBits:   bits,
		Signature:         aggregatedSig,
	}

	return &ethpbv2.ProduceSyncCommitteeContributionResponse{
		Data: contribution,
	}, nil
}

// SubmitContributionAndProofs publishes multiple signed sync committee contribution and proofs.
func (vs *Server) SubmitContributionAndProofs(ctx context.Context, req *ethpbv2.SubmitContributionAndProofsRequest) (*empty.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "validator.SubmitContributionAndProofs")
	defer span.End()

	for _, item := range req.Data {
		v1alpha1Req := &ethpbalpha.SignedContributionAndProof{
			Message: &ethpbalpha.ContributionAndProof{
				AggregatorIndex: item.Message.AggregatorIndex,
				Contribution: &ethpbalpha.SyncCommitteeContribution{
					Slot:              item.Message.Contribution.Slot,
					BlockRoot:         item.Message.Contribution.BeaconBlockRoot,
					SubcommitteeIndex: item.Message.Contribution.SubcommitteeIndex,
					AggregationBits:   item.Message.Contribution.AggregationBits,
					Signature:         item.Message.Contribution.Signature,
				},
				SelectionProof: item.Message.SelectionProof,
			},
			Signature: item.Signature,
		}
		_, err := vs.V1Alpha1Server.SubmitSignedContributionAndProof(ctx, v1alpha1Req)
		// We simply return err because it's already of a gRPC error type.
		if err != nil {
			return nil, err
		}
	}

	return &empty.Empty{}, nil
}

// attestationDependentRoot is get_block_root_at_slot(state, compute_start_slot_at_epoch(epoch - 1) - 1)
// or the genesis block root in the case of underflow.
func attestationDependentRoot(s state.BeaconState, epoch types.Epoch) ([]byte, error) {
	var dependentRootSlot types.Slot
	if epoch <= 1 {
		dependentRootSlot = 0
	} else {
		prevEpochStartSlot, err := slots.EpochStart(epoch.Sub(1))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not obtain epoch's start slot: %v", err)
		}
		dependentRootSlot = prevEpochStartSlot.Sub(1)
	}
	root, err := helpers.BlockRootAtSlot(s, dependentRootSlot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get block root")
	}
	return root, nil
}

// proposalDependentRoot is get_block_root_at_slot(state, compute_start_slot_at_epoch(epoch) - 1)
// or the genesis block root in the case of underflow.
func (vs *Server) proposalDependentRoot(ctx context.Context, s state.BeaconState, epoch types.Epoch) ([]byte, error) {
	var dependentRootSlot types.Slot
	if epoch == 0 {
		dependentRootSlot = 0
	} else {
		epochStartSlot, err := slots.EpochStart(epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not obtain epoch's start slot: %v", err)
		}
		dependentRootSlot = epochStartSlot.Sub(1)
	}
	var root []byte
	var err error
	// Per spec, if the dependent root epoch is greater than current epoch, use the head root.
	if dependentRootSlot >= s.Slot() {
		root, err = vs.HeadFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		root, err = helpers.BlockRootAtSlot(s, dependentRootSlot)
		if err != nil {
			return nil, errors.Wrap(err, "could not get block root")
		}
	}

	return root, nil
}

// advanceState advances state with empty transitions up to the requested epoch start slot.
// In case 1 epoch ahead was requested, we take the start slot of the current epoch.
// Taking the start slot of the next epoch would result in an error inside transition.ProcessSlots.
func advanceState(ctx context.Context, s state.BeaconState, requestedEpoch, currentEpoch types.Epoch) (state.BeaconState, error) {
	var epochStartSlot types.Slot
	var err error
	if requestedEpoch == currentEpoch+1 {
		epochStartSlot, err = slots.EpochStart(requestedEpoch.Sub(1))
		if err != nil {
			return nil, errors.Wrap(err, "Could not obtain epoch's start slot")
		}
	} else {
		epochStartSlot, err = slots.EpochStart(requestedEpoch)
		if err != nil {
			return nil, errors.Wrap(err, "Could not obtain epoch's start slot")
		}
	}
	s, err = transition.ProcessSlotsIfPossible(ctx, s, epochStartSlot)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not process slots up to %d", epochStartSlot)
	}

	return s, nil
}

// Logic based on https://hackmd.io/ofFJ5gOmQpu1jjHilHbdQQ
func v1ValidatorStatusToV1Alpha1(valStatus ethpbv1.ValidatorStatus) ethpbalpha.ValidatorStatus {
	switch valStatus {
	case ethpbv1.ValidatorStatus_ACTIVE:
		return ethpbalpha.ValidatorStatus_ACTIVE
	case ethpbv1.ValidatorStatus_PENDING:
		return ethpbalpha.ValidatorStatus_PENDING
	case ethpbv1.ValidatorStatus_WITHDRAWAL:
		return ethpbalpha.ValidatorStatus_EXITED
	default:
		return ethpbalpha.ValidatorStatus_UNKNOWN_STATUS
	}
}

func syncCommitteeDutiesLastValidEpoch(currentEpoch types.Epoch) types.Epoch {
	currentSyncPeriodIndex := currentEpoch / params.BeaconConfig().EpochsPerSyncCommitteePeriod
	// Return the last epoch of the next sync committee.
	// To do this we go two periods ahead to find the first invalid epoch, and then subtract 1.
	return (currentSyncPeriodIndex+2)*params.BeaconConfig().EpochsPerSyncCommitteePeriod - 1
}

func syncCommitteeDuties(
	valIndices []types.ValidatorIndex,
	st state.BeaconState,
	committeePubkeys map[[fieldparams.BLSPubkeyLength]byte][]uint64,
) ([]*ethpbv2.SyncCommitteeDuty, error) {
	duties := make([]*ethpbv2.SyncCommitteeDuty, 0)
	for _, index := range valIndices {
		duty := &ethpbv2.SyncCommitteeDuty{
			ValidatorIndex: index,
		}
		valPubkey48 := st.PubkeyAtIndex(index)
		zeroPubkey := [fieldparams.BLSPubkeyLength]byte{}
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
