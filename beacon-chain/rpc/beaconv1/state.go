package beaconv1

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetGenesis retrieves details of the chain's genesis which can be used to identify chain.
func (bs *Server) GetGenesis(ctx context.Context, _ *ptypes.Empty) (*ethpb.GenesisResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetGenesis")
	defer span.End()

	genesisTime := bs.GenesisTimeFetcher.GenesisTime()
	if genesisTime.IsZero() {
		return nil, status.Errorf(codes.NotFound, "Chain genesis info is not yet known")
	}
	validatorRoot := bs.ChainInfoFetcher.GenesisValidatorRoot()
	if bytes.Equal(validatorRoot[:], params.BeaconConfig().ZeroHash[:]) {
		return nil, status.Errorf(codes.NotFound, "Chain genesis info is not yet known")
	}
	forkVersion := params.BeaconConfig().GenesisForkVersion

	return &ethpb.GenesisResponse{
		Data: &ethpb.GenesisResponse_Genesis{
			GenesisTime: &ptypes.Timestamp{
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
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetStateRoot")
	defer span.End()

	var (
		stateIdString = strings.ToLower(string(req.StateId))
		root          []byte
		err           error
	)

	switch stateIdString {
	case "head":
		root, err = bs.headStateRoot(ctx)
	case "genesis":
		root, err = bs.genesisStateRoot(ctx)
	case "finalized":
		root, err = bs.finalizedStateRoot(ctx)
	case "justified":
		root, err = bs.justifiedStateRoot(ctx)
	default:
		ok, matchErr := bytesutil.IsBytes32Hex(req.StateId)
		if matchErr != nil {
			return nil, status.Errorf(codes.Internal, "Failed to parse ID: %v", err)
		}
		if ok {
			root, err = bs.stateRootByHex(ctx, req.StateId)
		} else {
			slot, parseErr := strconv.ParseUint(stateIdString, 10, 64)
			if parseErr != nil {
				// ID format does not match any valid options.
				return nil, status.Errorf(codes.Internal, "Invalid state ID: "+stateIdString)
			}
			root, err = bs.stateRootBySlot(ctx, slot)
		}
	}

	if err != nil {
		return nil, err
	}
	return &ethpb.StateRootResponse{
		Data: &ethpb.StateRootResponse_StateRoot{
			StateRoot: root,
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

func (bs *Server) headStateRoot(ctx context.Context) ([]byte, error) {
	stateRoot, err := bs.ChainInfoFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
	}
	return stateRoot, nil
}

func (bs *Server) genesisStateRoot(ctx context.Context) ([]byte, error) {
	stateRoot, err := bs.ChainStartFetcher.PreGenesisState().HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
	}
	return stateRoot[:], nil
}

func (bs *Server) finalizedStateRoot(ctx context.Context) ([]byte, error) {
	var blockRoot [32]byte
	copy(blockRoot[:], bs.ChainInfoFetcher.FinalizedCheckpt().Root)
	state, err := bs.StateGenService.StateByRoot(ctx, blockRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain state: %v", err)
	}
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
	}
	return stateRoot[:], nil
}

func (bs *Server) justifiedStateRoot(ctx context.Context) ([]byte, error) {
	var blockRoot [32]byte
	copy(blockRoot[:], bs.ChainInfoFetcher.CurrentJustifiedCheckpt().Root)
	state, err := bs.StateGenService.StateByRoot(ctx, blockRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain state: %v", err)
	}
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
	}
	return stateRoot[:], nil
}

func (bs *Server) stateRootByHex(ctx context.Context, stateId []byte) ([]byte, error) {
	var stateRoot [32]byte
	copy(stateRoot[:], stateId)
	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain head state: %v", err)
	}
	for _, root := range headState.StateRoots() {
		if bytes.Equal(root, stateRoot[:]) {
			return stateRoot[:], nil
		}
	}
	return nil, status.Errorf(
		codes.NotFound,
		"State not found in the last %d states", len(headState.StateRoots()))
}

func (bs *Server) stateRootBySlot(ctx context.Context, slot uint64) ([]byte, error) {
	currentSlot := bs.ChainInfoFetcher.HeadSlot()
	if slot > currentSlot {
		return nil, status.Errorf(codes.Internal, "Slot cannot be in the future")
	}
	state, err := bs.StateGenService.StateBySlot(ctx, slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain state: %v", err)
	}
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to obtain root: %v", err)
	}
	return stateRoot[:], nil
}
