package validator

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
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

	// An optimistic validator MUST NOT produce a block (i.e., sign across the DOMAIN_BEACON_PROPOSER domain).
	if err := vs.optimisticStatus(ctx); err != nil {
		return nil, err
	}

	return vs.getBellatrixBeaconBlock(ctx, req)
}

// GetBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot.
//
// DEPRECATED: Use GetBeaconBlock instead to handle blocks pre and post-Altair hard fork. This endpoint
// cannot handle blocks after the Altair fork epoch. If requesting a block after Altair, nothing will
// be returned.
func (vs *Server) GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.GetBlock")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))
	blk, err := vs.GetBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
	}
	return blk.GetPhase0(), nil
}

// ProposeBeaconBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (vs *Server) ProposeBeaconBlock(ctx context.Context, req *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.ProposeBeaconBlock")
	defer span.End()
	blk, err := wrapper.WrappedSignedBeaconBlock(req.Block)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not decode block: %v", err)
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
	blk, err := wrapper.WrappedSignedBeaconBlock(rBlk)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not decode block: %v", err)
	}
	return vs.proposeGenericBeaconBlock(ctx, blk)
}

// PrepareBeaconProposer caches and updates the fee recipient for the given proposer.
func (vs *Server) PrepareBeaconProposer(
	ctx context.Context, request *ethpb.PrepareBeaconProposerRequest,
) (*emptypb.Empty, error) {
	_, span := trace.StartSpan(ctx, "validator.PrepareBeaconProposer")
	defer span.End()
	var feeRecipients []common.Address
	var validatorIndices []types.ValidatorIndex
	for _, recipientContainer := range request.Recipients {
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
	log.WithFields(logrus.Fields{
		"validatorIndices": validatorIndices,
	}).Info("Updated fee recipient addresses for validator indices")
	return &emptypb.Empty{}, nil
}

func (vs *Server) proposeGenericBeaconBlock(ctx context.Context, blk interfaces.SignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.proposeGenericBeaconBlock")
	defer span.End()

	blk, err := vs.getBuilderBlock(ctx, blk)
	if err != nil {
		return nil, err
	}

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

func (vs *Server) getBuilderBlock(ctx context.Context, b interfaces.SignedBeaconBlock) (interfaces.SignedBeaconBlock, error) {
	if b.Version() != version.BellatrixBlind {
		return b, nil
	}
	if vs.BlockBuilder.Status() != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Block builder is not running")
	}
	agg, err := b.Block().Body().SyncAggregate()
	if err != nil {
		return nil, err
	}
	h, err := b.Block().Body().ExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	sb := &ethpb.SignedBlindedBeaconBlockBellatrix{
		Block: &ethpb.BlindedBeaconBlockBellatrix{
			Slot:          b.Block().Slot(),
			ProposerIndex: b.Block().ProposerIndex(),
			ParentRoot:    b.Block().ParentRoot(),
			StateRoot:     b.Block().StateRoot(),
			Body: &ethpb.BlindedBeaconBlockBodyBellatrix{
				RandaoReveal:           b.Block().Body().RandaoReveal(),
				Eth1Data:               b.Block().Body().Eth1Data(),
				Graffiti:               b.Block().Body().Graffiti(),
				ProposerSlashings:      b.Block().Body().ProposerSlashings(),
				AttesterSlashings:      b.Block().Body().AttesterSlashings(),
				Attestations:           b.Block().Body().Attestations(),
				Deposits:               b.Block().Body().Deposits(),
				VoluntaryExits:         b.Block().Body().VoluntaryExits(),
				SyncAggregate:          agg,
				ExecutionPayloadHeader: h,
			},
		},
		Signature: nil,
	}

	payload, err := vs.BlockBuilder.SubmitBlindedBlock(ctx, sb)
	if err != nil {
		return nil, err
	}
	bb := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot:          sb.Block.Slot,
			ProposerIndex: sb.Block.ProposerIndex,
			ParentRoot:    sb.Block.ParentRoot,
			StateRoot:     sb.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				RandaoReveal:      sb.Block.Body.RandaoReveal,
				Eth1Data:          sb.Block.Body.Eth1Data,
				Graffiti:          sb.Block.Body.Graffiti,
				ProposerSlashings: sb.Block.Body.ProposerSlashings,
				AttesterSlashings: sb.Block.Body.AttesterSlashings,
				Attestations:      sb.Block.Body.Attestations,
				Deposits:          sb.Block.Body.Deposits,
				VoluntaryExits:    sb.Block.Body.VoluntaryExits,
				SyncAggregate:     agg,
				ExecutionPayload:  payload,
			},
		},
		Signature: nil,
	}
	wb, err := wrapper.WrappedSignedBeaconBlock(bb)
	if err != nil {
		return nil, err
	}
	return wb, nil
}

// computeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (vs *Server) computeStateRoot(ctx context.Context, block interfaces.SignedBeaconBlock) ([]byte, error) {
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
