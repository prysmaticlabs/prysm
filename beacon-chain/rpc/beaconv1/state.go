package beaconv1

import (
	"context"
	"strconv"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetGenesis retrieves details of the chain's genesis which can be used to identify chain.
func (bs *Server) GetGenesis(ctx context.Context, _ *ptypes.Empty) (*ethpb.GenesisResponse, error) {
	genesisTime := bs.GenesisTimeFetcher.GenesisTime()
	var defaultGenesisTime time.Time
	var gt *ptypes.Timestamp
	var err error
	if genesisTime == defaultGenesisTime {
		gt, err = ptypes.TimestampProto(time.Unix(0, 0))
	} else {
		gt, err = ptypes.TimestampProto(genesisTime)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert genesis time to proto: %v", err)
	}

	genValRoot := bs.ChainInfoFetcher.GenesisValidatorRoot()
	return &ethpb.GenesisResponse{
		GenesisTime:           gt,
		GenesisValidatorsRoot: genValRoot[:],
		GenesisForkVersion:    params.BeaconConfig().GenesisForkVersion,
	}, nil
}

// GetStateRoot calculates HashTreeRoot for state with given 'stateId'. If stateId is root, same value will be returned.
func (bs *Server) GetStateRoot(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateRootResponse, error) {
	requestedState, err := bs.getState(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state from ID: %v", err)
	}
	stateRoot, err := requestedState.HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state root: %v", err)
	}
	return &ethpb.StateRootResponse{
		StateRoot: stateRoot[:],
	}, nil
}

// GetStateFork returns Fork object for state with given 'stateId'.
func (bs *Server) GetStateFork(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateForkResponse, error) {
	requestedState, err := bs.getState(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state from ID: %f", err)
	}
	fork := requestedState.Fork()
	return &ethpb.StateForkResponse{
		Fork: &ethpb.Fork{
			PreviousVersion: fork.PreviousVersion,
			CurrentVersion:  fork.CurrentVersion,
			Epoch:           fork.Epoch,
		},
	}, nil
}

// GetFinalityCheckpoints returns finality checkpoints for state with given 'stateId'. In case finality is
// not yet achieved, checkpoint should return epoch 0 and ZERO_HASH as root.
func (bs *Server) GetFinalityCheckpoints(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateFinalityCheckpointResponse, error) {
	requestedState, err := bs.getState(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state from ID: %v", err)
	}
	prevJustChkpt := requestedState.PreviousJustifiedCheckpoint()
	curJustChkpt := requestedState.CurrentJustifiedCheckpoint()
	finalizedChkpt := requestedState.FinalizedCheckpoint()
	resp := &ethpb.StateFinalityCheckpointResponse{
		PreviousJustified: &ethpb.Checkpoint{
			Epoch: prevJustChkpt.Epoch,
			Root:  prevJustChkpt.Root,
		},
		CurrentJustified: &ethpb.Checkpoint{
			Epoch: curJustChkpt.Epoch,
			Root:  curJustChkpt.Root,
		},
		Finalized: &ethpb.Checkpoint{
			Epoch: finalizedChkpt.Epoch,
			Root:  finalizedChkpt.Root,
		},
	}
	return resp, nil
}

func (bs *Server) getState(ctx context.Context, stateId []byte) (*state.BeaconState, error) {
	var requestedState *state.BeaconState
	var err error
	switch string(stateId) {
	case "head":
		requestedState, err = bs.ChainInfoFetcher.HeadState(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get head state")
		}
	case "genesis":
		requestedState, err = bs.StateGen.StateByRoot(ctx, params.BeaconConfig().ZeroHash)
		if err != nil {
			return nil, errors.Wrap(err, "could not get genesis checkpoint")
		}
	case "finalized":
		finalizedCheckpoint := bs.ChainInfoFetcher.FinalizedCheckpt()
		requestedState, err = bs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(finalizedCheckpoint.Root))
		if err != nil {
			return nil, errors.Wrap(err, "could not get finalized checkpoint")
		}
	case "justified":
		justifiedCheckpoint := bs.ChainInfoFetcher.CurrentJustifiedCheckpt()
		requestedState, err = bs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(justifiedCheckpoint.Root))
		if err != nil {
			return nil, errors.Wrap(err, "could not get justified checkpoint")
		}
	default:
		if len(stateId) == 32 {
			requestedState, err = bs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(stateId))
			if err != nil {
				return nil, errors.Wrap(err, "could not get state")
			}
		} else {
			requestedSlot, err := strconv.ParseUint(string(stateId), 10, 64)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Cannot parse slot from input %#x: %v", stateId, err)
			}
			currentSlot := bs.ChainInfoFetcher.HeadSlot()
			if requestedSlot > currentSlot {
				return nil, status.Errorf(
					codes.InvalidArgument,
					"Cannot retrieve information about a slot in the future, current slot %d, requesting %d",
					currentSlot,
					requestedSlot,
				)
			}
			requestedEpoch := helpers.SlotToEpoch(requestedSlot)
			currentEpoch := helpers.SlotToEpoch(currentSlot)
			if requestedEpoch > currentEpoch {
				return nil, status.Errorf(
					codes.InvalidArgument,
					"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
					currentEpoch,
					requestedEpoch,
				)
			}

			requestedState, err = bs.StateGen.StateBySlot(ctx, requestedSlot)
			if err != nil {
				return nil, errors.Wrap(err, "could not get state")
			}
		}
	}
	if requestedState == nil {
		return nil, errors.New("could not find state with state id")
	}
	return requestedState, nil
}
