package validator

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// eth1DataNotification is a latch to stop flooding logs with the same warning.
var eth1DataNotification bool

const eth1dataTimeout = 2 * time.Second

// GetBeaconBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot. Returns phase0 beacon blocks
// before the Altair fork epoch and Altair blocks post-fork epoch.
func (vs *Server) GetBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.GetBeaconBlock")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	if slots.ToEpoch(req.Slot) < params.BeaconConfig().AltairForkEpoch {
		blk, err := vs.getPhase0BeaconBlock(ctx, req)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch phase0 beacon block: %v", err)
		}
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Phase0{Phase0: blk}}, nil
	} else if slots.ToEpoch(req.Slot) < params.BeaconConfig().BellatrixForkEpoch {
		blk, err := vs.getAltairBeaconBlock(ctx, req)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch Altair beacon block: %v", err)
		}
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Altair{Altair: blk}}, nil
	}

	blk, err := vs.getMergeBeaconBlock(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not fetch Merge beacon block: %v", err)
	}

	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Merge{Merge: blk}}, nil
}

// GetBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot.
//
// DEPRECATED: Use GetBeaconBlock instead to handle blocks pre and post-Altair hard fork. This endpoint
// cannot handle blocks after the Altair fork epoch.
func (vs *Server) GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.GetBlock")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))
	return vs.getPhase0BeaconBlock(ctx, req)
}

func (vs *Server) getMergeBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlockMerge, error) {
	altairBlk, err := vs.buildAltairBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
	}
	payload, err := vs.getExecutionPayload(ctx, req.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution payload")
	}

	log.WithFields(logrus.Fields{
		"blockNumber":   payload.BlockNumber,
		"blockHash":     fmt.Sprintf("%#x", payload.BlockHash),
		"parentHash":    fmt.Sprintf("%#x", payload.ParentHash),
		"coinBase":      fmt.Sprintf("%#x", payload.FeeRecipient),
		"gasLimit":      payload.GasLimit,
		"gasUsed":       payload.GasUsed,
		"baseFeePerGas": payload.BaseFeePerGas,
		"random":        fmt.Sprintf("%#x", payload.Random),
		"extraData":     fmt.Sprintf("%#x", payload.ExtraData),
		"txs":           payload.Transactions,
	}).Info("Retrieved payload")

	blk := &ethpb.BeaconBlockMerge{
		Slot:          altairBlk.Slot,
		ProposerIndex: altairBlk.ProposerIndex,
		ParentRoot:    altairBlk.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		Body: &ethpb.BeaconBlockBodyMerge{
			RandaoReveal:      altairBlk.Body.RandaoReveal,
			Eth1Data:          altairBlk.Body.Eth1Data,
			Graffiti:          altairBlk.Body.Graffiti,
			ProposerSlashings: altairBlk.Body.ProposerSlashings,
			AttesterSlashings: altairBlk.Body.AttesterSlashings,
			Attestations:      altairBlk.Body.Attestations,
			Deposits:          altairBlk.Body.Deposits,
			VoluntaryExits:    altairBlk.Body.VoluntaryExits,
			SyncAggregate:     altairBlk.Body.SyncAggregate,
			ExecutionPayload:  payload,
		},
	}
	// Compute state root with the newly constructed block.
	wsb, err := wrapper.WrappedMergeSignedBeaconBlock(
		&ethpb.SignedBeaconBlockMerge{Block: blk, Signature: make([]byte, 96)},
	)
	if err != nil {
		return nil, err
	}
	stateRoot, err := vs.computeStateRoot(ctx, wsb)
	if err != nil {
		interop.WriteBlockToDisk(wsb, true /*failed*/)
		return nil, fmt.Errorf("could not compute state root: %v", err)
	}
	blk.StateRoot = stateRoot
	return blk, nil
}

func (vs *Server) buildAltairBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlockAltair, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.buildAltairBeaconBlock")
	defer span.End()
	blkData, err := vs.buildPhase0BlockData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not build block data: %v", err)
	}
	// Use zero hash as stub for state root to compute later.
	stateRoot := params.BeaconConfig().ZeroHash[:]

	// No need for safe sub as req.Slot cannot be 0 if requesting Altair blocks. If 0, we will be throwing
	// an error in the first validity check of this endpoint.
	syncAggregate, err := vs.getSyncAggregate(ctx, req.Slot-1, bytesutil.ToBytes32(blkData.ParentRoot))
	if err != nil {
		return nil, err
	}

	return &ethpb.BeaconBlockAltair{
		Slot:          req.Slot,
		ParentRoot:    blkData.ParentRoot,
		StateRoot:     stateRoot,
		ProposerIndex: blkData.ProposerIdx,
		Body: &ethpb.BeaconBlockBodyAltair{
			Eth1Data:          blkData.Eth1Data,
			Deposits:          blkData.Deposits,
			Attestations:      blkData.Attestations,
			RandaoReveal:      req.RandaoReveal,
			ProposerSlashings: blkData.ProposerSlashings,
			AttesterSlashings: blkData.AttesterSlashings,
			VoluntaryExits:    blkData.VoluntaryExits,
			Graffiti:          blkData.Graffiti[:],
			SyncAggregate:     syncAggregate,
		},
	}, nil
}

// ProposeBeaconBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (vs *Server) ProposeBeaconBlock(ctx context.Context, req *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.ProposeBeaconBlock")
	defer span.End()
	var blk block.SignedBeaconBlock
	var err error
	switch b := req.Block.(type) {
	case *ethpb.GenericSignedBeaconBlock_Phase0:
		blk = wrapper.WrappedPhase0SignedBeaconBlock(b.Phase0)
	case *ethpb.GenericSignedBeaconBlock_Altair:
		blk, err = wrapper.WrappedAltairSignedBeaconBlock(b.Altair)
		if err != nil {
			return nil, status.Error(codes.Internal, "could not wrap altair beacon block")
		}
	case *ethpb.GenericSignedBeaconBlock_Merge:
		blk, err = wrapper.WrappedMergeSignedBeaconBlock(b.Merge)
		if err != nil {
			return nil, status.Error(codes.Internal, "could not wrap merge beacon block")
		}
	default:
		return nil, status.Error(codes.Internal, "block version not supported")
	}
	return vs.proposeGenericBeaconBlock(ctx, blk)
}

// ProposeBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
//
// DEPRECATED: Use ProposeBeaconBlock instead.
func (vs *Server) ProposeBlock(ctx context.Context, rBlk *ethpb.SignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.ProposeBlock")
	defer span.End()
	blk := wrapper.WrappedPhase0SignedBeaconBlock(rBlk)
	return vs.proposeGenericBeaconBlock(ctx, blk)
}

func (vs *Server) proposeGenericBeaconBlock(ctx context.Context, blk block.SignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.proposeGenericBeaconBlock")
	defer span.End()
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("could not tree hash block: %v", err)
	}

	// Do not block proposal critical path with debug logging or block feed updates.
	defer func() {
		log.WithField("blockRoot", fmt.Sprintf("%#x", bytesutil.Trunc(root[:]))).Debugf(
			"Block proposal received via RPC")
		vs.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: &blockfeed.ReceivedBlockData{SignedBlock: blk},
		})
	}()

	// Broadcast the new block to the network.
	if err := vs.P2P.Broadcast(ctx, blk.Proto()); err != nil {
		return nil, fmt.Errorf("could not broadcast block: %v", err)
	}
	log.WithFields(logrus.Fields{
		"blockRoot": hex.EncodeToString(root[:]),
	}).Debug("Broadcasting block")

	if err := vs.BlockReceiver.ReceiveBlock(ctx, blk, root); err != nil {
		return nil, fmt.Errorf("could not process beacon block: %v", err)
	}

	return &ethpb.ProposeResponse{
		BlockRoot: root[:],
	}, nil
}

// computeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (vs *Server) computeStateRoot(ctx context.Context, block block.SignedBeaconBlock) ([]byte, error) {
	beaconState, err := vs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(block.Block().ParentRoot()))
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve beacon state")
	}
	root, err := transition.CalculateStateRoot(
		ctx,
		beaconState,
		block,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate state root at slot %d", beaconState.Slot())
	}

	log.WithField("beaconStateRoot", fmt.Sprintf("%#x", root)).Debugf("Computed state root")
	return root[:], nil
}
