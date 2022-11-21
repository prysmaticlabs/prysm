package beacon

import (
	"bytes"
	"context"
	"strconv"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/statefetcher"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	eth2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stateRequest struct {
	epoch   *types.Epoch
	stateId []byte
}

// GetGenesis retrieves details of the chain's genesis which can be used to identify chain.
func (bs *Server) GetGenesis(ctx context.Context, _ *emptypb.Empty) (*ethpb.GenesisResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetGenesis")
	defer span.End()

	genesisTime := bs.GenesisTimeFetcher.GenesisTime()
	if genesisTime.IsZero() {
		return nil, status.Errorf(codes.NotFound, "Chain genesis info is not yet known")
	}
	validatorRoot := bs.ChainInfoFetcher.GenesisValidatorsRoot()
	if bytes.Equal(validatorRoot[:], params.BeaconConfig().ZeroHash[:]) {
		return nil, status.Errorf(codes.NotFound, "Chain genesis info is not yet known")
	}
	forkVersion := params.BeaconConfig().GenesisForkVersion

	return &ethpb.GenesisResponse{
		Data: &ethpb.GenesisResponse_Genesis{
			GenesisTime: &timestamppb.Timestamp{
				Seconds: genesisTime.Unix(),
				Nanos:   0,
			},
			GenesisValidatorsRoot: validatorRoot[:],
			GenesisForkVersion:    forkVersion,
		},
	}, nil
}

// GetStateRoot calculates HashTreeRoot for state with given 'stateId'. If stateId is root, same value will be returned.
func (bs *Server) GetStateRoot(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateRootResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetStateRoot")
	defer span.End()

	stateRoot, err := bs.StateFetcher.StateRoot(ctx, req.StateId)
	if err != nil {
		if rootNotFoundErr, ok := err.(*statefetcher.StateRootNotFoundError); ok {
			return nil, status.Errorf(codes.NotFound, "State root not found: %v", rootNotFoundErr)
		} else if parseErr, ok := err.(*statefetcher.StateIdParseError); ok {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid state ID: %v", parseErr)
		}
		return nil, status.Errorf(codes.Internal, "Could not get state root: %v", err)
	}
	st, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}
	isOptimistic, err := helpers.IsOptimistic(ctx, st, bs.OptimisticModeFetcher)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	return &ethpb.StateRootResponse{
		Data: &ethpb.StateRootResponse_StateRoot{
			Root: stateRoot,
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

// GetStateFork returns Fork object for state with given 'stateId'.
func (bs *Server) GetStateFork(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateForkResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetStateFork")
	defer span.End()

	st, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}
	fork := st.Fork()
	isOptimistic, err := helpers.IsOptimistic(ctx, st, bs.OptimisticModeFetcher)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	return &ethpb.StateForkResponse{
		Data: &ethpb.Fork{
			PreviousVersion: fork.PreviousVersion,
			CurrentVersion:  fork.CurrentVersion,
			Epoch:           fork.Epoch,
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

// GetFinalityCheckpoints returns finality checkpoints for state with given 'stateId'. In case finality is
// not yet achieved, checkpoint should return epoch 0 and ZERO_HASH as root.
func (bs *Server) GetFinalityCheckpoints(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateFinalityCheckpointResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetFinalityCheckpoints")
	defer span.End()

	st, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}
	isOptimistic, err := helpers.IsOptimistic(ctx, st, bs.OptimisticModeFetcher)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	return &ethpb.StateFinalityCheckpointResponse{
		Data: &ethpb.StateFinalityCheckpointResponse_StateFinalityCheckpoint{
			PreviousJustified: checkpoint(st.PreviousJustifiedCheckpoint()),
			CurrentJustified:  checkpoint(st.CurrentJustifiedCheckpoint()),
			Finalized:         checkpoint(st.FinalizedCheckpoint()),
		},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

// GetRandao fetches the RANDAO mix for the requested epoch from the state identified by state_id.
// If an epoch is not specified then the RANDAO mix for the state's current epoch will be returned.
// By adjusting the state_id parameter you can query for any historic value of the RANDAO mix.
// Ordinarily states from the same epoch will mutate the RANDAO mix for that epoch as blocks are applied.
func (bs *Server) GetRandao(ctx context.Context, req *eth2.RandaoRequest) (*eth2.RandaoResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.GetRandao")
	defer span.End()

	st, err := bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	stEpoch := slots.ToEpoch(st.Slot())
	epoch := stEpoch
	if req.Epoch != nil {
		epoch = *req.Epoch
	}

	// future epochs and epochs too far back are not supported.
	randaoEpochLowerBound := uint64(0)
	// Lower bound should not underflow.
	if uint64(stEpoch) > uint64(st.RandaoMixesLength()) {
		randaoEpochLowerBound = uint64(stEpoch) - uint64(st.RandaoMixesLength())
	}
	if epoch > stEpoch || uint64(epoch) < randaoEpochLowerBound+1 {
		return nil, status.Errorf(codes.InvalidArgument, "Epoch is out of range for the randao mixes of the state")
	}
	idx := epoch % params.BeaconConfig().EpochsPerHistoricalVector
	randao, err := st.RandaoMixAtIndex(uint64(idx))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get randao mix at index %d", idx)
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, st, bs.OptimisticModeFetcher)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	return &eth2.RandaoResponse{
		Data:                &eth2.RandaoResponse_Randao{Randao: randao},
		ExecutionOptimistic: isOptimistic,
	}, nil
}

func (bs *Server) stateFromRequest(ctx context.Context, req *stateRequest) (state.BeaconState, error) {
	if req.epoch != nil {
		slot, err := slots.EpochStart(*req.epoch)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not calculate start slot for epoch %d: %v",
				*req.epoch,
				err,
			)
		}
		st, err := bs.StateFetcher.State(ctx, []byte(strconv.FormatUint(uint64(slot), 10)))
		if err != nil {
			return nil, helpers.PrepareStateFetchGRPCError(err)
		}
		return st, nil
	}
	var err error
	st, err := bs.StateFetcher.State(ctx, req.stateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}
	return st, nil
}

func checkpoint(sourceCheckpoint *eth.Checkpoint) *ethpb.Checkpoint {
	if sourceCheckpoint != nil {
		return &ethpb.Checkpoint{
			Epoch: sourceCheckpoint.Epoch,
			Root:  sourceCheckpoint.Root,
		}
	}
	return &ethpb.Checkpoint{
		Epoch: 0,
		Root:  params.BeaconConfig().ZeroHash[:],
	}
}
