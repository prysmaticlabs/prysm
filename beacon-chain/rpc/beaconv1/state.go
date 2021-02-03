package beaconv1

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetGenesis retrieves details of the chain's genesis which can be used to identify chain.
func (bs *Server) GetGenesis(ctx context.Context, _ *ptypes.Empty) (*ethpb.GenesisResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetStateRoot calculates HashTreeRoot for state with given 'stateId'. If stateId is root, same value will be returned.
func (bs *Server) GetStateRoot(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateRootResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetStateRoot")
	defer span.End()

	stateIdString := string(req.StateId)
	// TODO: Extract to helper functions, add comments
	if stateIdString == "head" {
		stateRoot, err := bs.ChainInfoFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
		}
		return &ethpb.StateRootResponse{
			Data: &ethpb.StateRootResponse_StateRoot{
				StateRoot: stateRoot,
			},
		}, nil
	}
	if stateIdString == "genesis" {
		stateRoot, err := bs.ChainStartFetcher.PreGenesisState().HashTreeRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
		}
		return &ethpb.StateRootResponse{
			Data: &ethpb.StateRootResponse_StateRoot{
				StateRoot: stateRoot[:],
			},
		}, nil
	}
	if stateIdString == "finalized" {
		var blockRoot [32]byte
		copy(blockRoot[:], bs.ChainInfoFetcher.FinalizedCheckpt().Root)
		state, err := bs.StateGen.StateByRoot(ctx, blockRoot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to obtain state: %v", err)
		}
		stateRoot, err := state.HashTreeRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
		}
		return &ethpb.StateRootResponse{
			Data: &ethpb.StateRootResponse_StateRoot{
				StateRoot: stateRoot[:],
			},
		}, nil
	}
	if stateIdString == "justified" {
		var blockRoot [32]byte
		copy(blockRoot[:], bs.ChainInfoFetcher.CurrentJustifiedCheckpt().Root)
		state, err := bs.StateGen.StateByRoot(ctx, blockRoot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to obtain state: %v", err)
		}
		stateRoot, err := state.HashTreeRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
		}
		return &ethpb.StateRootResponse{
			Data: &ethpb.StateRootResponse_StateRoot{
				StateRoot: stateRoot[:],
			},
		}, nil
	}

	ok, err := regexp.Match("0x[0-9a-fA-F]{64}", []byte(hexutil.Encode(req.StateId)))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to parse ID: %v", err)
	}
	if ok {
		var stateRoot [32]byte
		copy(stateRoot[:], req.StateId)
		headState, err := bs.ChainInfoFetcher.HeadState(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to obtain head state: %v", err)
		}
		for _, root := range headState.StateRoots() {
			if bytes.Equal(root, stateRoot[:]) {
				log.Error(string(root))
				return &ethpb.StateRootResponse{
					Data: &ethpb.StateRootResponse_StateRoot{
						StateRoot: stateRoot[:],
					},
				}, nil
			}
		}
		return nil, status.Errorf(
			codes.NotFound,
			"State not found in the last %d states", len(headState.StateRoots()))
	}

	slot, err := strconv.ParseUint(stateIdString, 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Invalid state ID: "+stateIdString)
	}
	currentSlot := bs.ChainInfoFetcher.HeadSlot()
	if slot < 0 || slot > currentSlot {
		return nil, status.Errorf(codes.Internal, "Slot has to be between 0 and %d", currentSlot)
	}
	state, err := bs.StateGen.StateBySlot(ctx, slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain state: %v", err)
	}
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
	}
	return &ethpb.StateRootResponse{
		Data: &ethpb.StateRootResponse_StateRoot{
			StateRoot: stateRoot[:],
		},
	}, nil
}

// GetStateFork returns Fork object for state with given 'stateId'.
func (bs *Server) GetStateFork(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateForkResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetFinalityCheckpoints returns finality checkpoints for state with given 'stateId'. In case finality is
// not yet achieved, checkpoint should return epoch 0 and ZERO_HASH as root.
func (bs *Server) GetFinalityCheckpoints(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateFinalityCheckpointResponse, error) {
	return nil, errors.New("unimplemented")
}
