package beaconv1

import (
	"context"
	"encoding/hex"
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

	genValRoot := bs.GenesisFetcher.GenesisValidatorRoot()
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
		return nil, status.Errorf(codes.Internal, "Could not get state from ID")
	}
	stateRoot, err := requestedState.HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state root")
	}
	return &ethpb.StateRootResponse{
		StateRoot: stateRoot[:],
	}, errors.New("unimplemented")
}

// GetStateFork returns Fork object for state with given 'stateId'.
func (bs *Server) GetStateFork(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateForkResponse, error) {
	requestedState, err := bs.getState(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state from ID")
	}
	fork := requestedState.Fork()
	return &ethpb.StateForkResponse{
		Fork: &ethpb.Fork{
			PreviousVersion: fork.PreviousVersion,
			CurrentVersion:  fork.CurrentVersion,
			Epoch:           helpers.SlotToEpoch(requestedState.Slot()),
		},
	}, nil
}

// GetFinalityCheckpoints returns finality checkpoints for state with given 'stateId'. In case finality is
// not yet achieved, checkpoint should return epoch 0 and ZERO_HASH as root.
func (bs *Server) GetFinalityCheckpoints(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateFinalityCheckpointResponse, error) {
	requestedState, err := bs.getState(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state from ID")
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

func (bs *Server) getState(ctx context.Context, stateId string) (*state.BeaconState, error) {
	switch stateId {
	case "head":
		headState, err := bs.HeadFetcher.HeadState(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get head state")
		}
		return headState, nil
	case "genesis":
		genesisState, err := bs.StateGen.StateByRoot(ctx, params.BeaconConfig().ZeroHash)
		if err != nil {
			return nil, errors.Wrap(err, "could not get genesis checkpoint")
		}
		return genesisState, nil
	case "finalized":
		finalizedCheckpoint := bs.FinalizationFetcher.FinalizedCheckpt()
		finalizedState, err := bs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(finalizedCheckpoint.Root))
		if err != nil {
			return nil, errors.Wrap(err, "could not get finalized checkpoint")
		}
		return finalizedState, nil
	case "justified":
		justifiedCheckpoint := bs.FinalizationFetcher.CurrentJustifiedCheckpt()
		justifiedState, err := bs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(justifiedCheckpoint.Root))
		if err != nil {
			return nil, errors.Wrap(err, "could not get justified checkpoint")
		}
		return justifiedState, nil
	default:
		parsed, err := hex.DecodeString(stateId)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse string")
		}
		if len(parsed) == 32 {
			requestedState, err := bs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(parsed))
			if err != nil {
				return nil, errors.Wrap(err, "could not get state")
			}
			return requestedState, nil
		} else {
			requestedSlot := bytesutil.FromBytes8(parsed)
			requestedEpoch := helpers.SlotToEpoch(requestedSlot)
			currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
			if requestedEpoch > currentEpoch {
				return nil, status.Errorf(
					codes.InvalidArgument,
					"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
					currentEpoch,
					requestedEpoch,
				)
			}

			requestedState, err := bs.StateGen.StateBySlot(ctx, requestedSlot)
			if err != nil {
				return nil, errors.Wrap(err, "could not get state")
			}
			return requestedState, nil
		}
	}
}
