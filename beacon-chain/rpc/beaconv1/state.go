package beaconv1

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
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
		root []byte
		err  error
	)

	root, err = bs.stateRoot(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state root: %v", err)
	}

	return &ethpb.StateRootResponse{
		Data: &ethpb.StateRootResponse_StateRoot{
			StateRoot: root,
		},
	}, nil
}

// GetStateFork returns Fork object for state with given 'stateId'.
func (bs *Server) GetStateFork(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateForkResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetStateFork")
	defer span.End()

	var (
		state iface.BeaconState
		err   error
	)

	state, err = bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}

	fork := state.Fork()
	return &ethpb.StateForkResponse{
		Data: &ethpb.Fork{
			PreviousVersion: fork.PreviousVersion,
			CurrentVersion:  fork.CurrentVersion,
			Epoch:           fork.Epoch,
		},
	}, nil
}

// GetFinalityCheckpoints returns finality checkpoints for state with given 'stateId'. In case finality is
// not yet achieved, checkpoint should return epoch 0 and ZERO_HASH as root.
func (bs *Server) GetFinalityCheckpoints(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateFinalityCheckpointResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetFinalityCheckpoints")
	defer span.End()

	var (
		state iface.BeaconState
		err   error
	)

	state, err = bs.StateFetcher.State(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state: %v", err)
	}

	return &ethpb.StateFinalityCheckpointResponse{
		Data: &ethpb.StateFinalityCheckpointResponse_StateFinalityCheckpoint{
			PreviousJustified: checkpoint(state.PreviousJustifiedCheckpoint()),
			CurrentJustified:  checkpoint(state.CurrentJustifiedCheckpoint()),
			Finalized:         checkpoint(state.FinalizedCheckpoint()),
		},
	}, nil
}

func (bs *Server) stateRoot(ctx context.Context, stateId []byte) ([]byte, error) {
	var (
		root []byte
		err  error
	)

	stateIdString := strings.ToLower(string(stateId))
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
		ok, matchErr := bytesutil.IsBytes32Hex(stateId)
		if matchErr != nil {
			return nil, errors.Wrap(err, "could not parse ID")
		}
		if ok {
			root, err = bs.stateRootByHex(ctx, stateId)
		} else {
			slotNumber, parseErr := strconv.ParseUint(stateIdString, 10, 64)
			if parseErr != nil {
				// ID format does not match any valid options.
				return nil, errors.New("invalid state ID: " + stateIdString)
			}
			root, err = bs.stateRootBySlot(ctx, types.Slot(slotNumber))
		}
	}

	return root, err
}

func (bs *Server) headStateRoot(ctx context.Context) ([]byte, error) {
	b, err := bs.ChainInfoFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head block")
	}
	if err := helpers.VerifyNilBeaconBlock(b); err != nil {
		return nil, err
	}
	return b.Block.StateRoot, nil
}

func (bs *Server) genesisStateRoot(ctx context.Context) ([]byte, error) {
	b, err := bs.BeaconDB.GenesisBlock(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get genesis block")
	}
	if err := helpers.VerifyNilBeaconBlock(b); err != nil {
		return nil, err
	}
	return b.Block.StateRoot, nil
}

func (bs *Server) finalizedStateRoot(ctx context.Context) ([]byte, error) {
	cp, err := bs.BeaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get finalized checkpoint")
	}
	b, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(cp.Root))
	if err != nil {
		return nil, errors.Wrap(err, "could not get finalized block")
	}
	if err := helpers.VerifyNilBeaconBlock(b); err != nil {
		return nil, err
	}
	return b.Block.StateRoot, nil
}

func (bs *Server) justifiedStateRoot(ctx context.Context) ([]byte, error) {
	cp, err := bs.BeaconDB.JustifiedCheckpoint(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get justified checkpoint")
	}
	b, err := bs.BeaconDB.Block(ctx, bytesutil.ToBytes32(cp.Root))
	if err != nil {
		return nil, errors.Wrap(err, "could not get justified block")
	}
	if err := helpers.VerifyNilBeaconBlock(b); err != nil {
		return nil, err
	}
	return b.Block.StateRoot, nil
}

func (bs *Server) stateRootByHex(ctx context.Context, stateId []byte) ([]byte, error) {
	var stateRoot [32]byte
	copy(stateRoot[:], stateId)
	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get head state")
	}
	for _, root := range headState.StateRoots() {
		if bytes.Equal(root, stateRoot[:]) {
			return stateRoot[:], nil
		}
	}
	return nil, fmt.Errorf("state not found in the last %d state roots in head state", len(headState.StateRoots()))
}

func (bs *Server) stateRootBySlot(ctx context.Context, slot types.Slot) ([]byte, error) {
	currentSlot := bs.GenesisTimeFetcher.CurrentSlot()
	if slot > currentSlot {
		return nil, errors.New("slot cannot be in the future")
	}
	found, blks, err := bs.BeaconDB.BlocksBySlot(ctx, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get blocks")
	}
	if !found {
		return nil, errors.New("no block exists")
	}
	if len(blks) != 1 {
		return nil, errors.New("multiple blocks exist in same slot")
	}
	if blks[0] == nil || blks[0].Block == nil {
		return nil, errors.New("nil block")
	}
	return blks[0].Block.StateRoot, nil
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
