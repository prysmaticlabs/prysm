package validator

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/builder"
	blocks2 "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	v "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// eth1DataNotification is a latch to stop flooding logs with the same warning.
var eth1DataNotification bool

const eth1dataTimeout = 2 * time.Second

// GetBeaconBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot. Returns a full block
// corresponding to the fork epoch
func (vs *Server) GetBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.GetBeaconBlock")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	// A syncing validator should not produce a block.
	if vs.SyncChecker.Syncing() {
		return nil, fmt.Errorf("syncing to latest head, not ready to respond")
	}

	// An optimistic validator MUST NOT produce a block (i.e., sign across the DOMAIN_BEACON_PROPOSER domain).
	if err := vs.optimisticStatus(ctx); err != nil {
		return nil, err
	}

	var blk interfaces.BeaconBlock
	var sBlk interfaces.SignedBeaconBlock
	var err error
	switch {
	case slots.ToEpoch(req.Slot) < params.BeaconConfig().AltairForkEpoch:
		blk, err = blocks.NewBeaconBlock(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
	case slots.ToEpoch(req.Slot) < params.BeaconConfig().BellatrixForkEpoch:
		blk, err = blocks.NewBeaconBlock(&ethpb.BeaconBlockAltair{Body: &ethpb.BeaconBlockBodyAltair{}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{Body: &ethpb.BeaconBlockBodyAltair{}}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
	case slots.ToEpoch(req.Slot) < params.BeaconConfig().CapellaForkEpoch:
		blk, err = blocks.NewBeaconBlock(&ethpb.BeaconBlockBellatrix{Body: &ethpb.BeaconBlockBodyBellatrix{}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{Block: &ethpb.BeaconBlockBellatrix{Body: &ethpb.BeaconBlockBodyBellatrix{}}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
	default:
		blk, err = blocks.NewBeaconBlock(&ethpb.BeaconBlockCapella{Body: &ethpb.BeaconBlockBodyCapella{}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockCapella{Block: &ethpb.BeaconBlockCapella{Body: &ethpb.BeaconBlockBodyCapella{}}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
	}

	parentRoot, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve head root: %v", err)
	}
	head, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get head state %v", err)
	}
	head, err = transition.ProcessSlotsUsingNextSlotCache(ctx, head, parentRoot, req.Slot)
	if err != nil {
		return nil, fmt.Errorf("could not advance slots to calculate proposer index: %v", err)
	}

	// Set slot, graffiti, randao reveal, and parent root.
	blk.SetSlot(req.Slot)
	blk.Body().SetGraffiti(req.Graffiti)
	blk.Body().SetRandaoReveal(req.RandaoReveal)
	blk.SetParentRoot(parentRoot)
	// Set eth1 data.
	eth1Data, err := vs.eth1DataMajorityVote(ctx, head)
	if err != nil {
		return nil, fmt.Errorf("could not get ETH1 data: %v", err)
	}
	blk.Body().SetEth1Data(eth1Data)

	// Set deposit and attestation.
	deposits, atts, err := vs.packDepositsAndAttestations(ctx, head, eth1Data)
	if err != nil {
		return nil, err
	}
	blk.Body().SetDeposits(deposits)
	blk.Body().SetAttestations(atts)

	// Set proposer index
	idx, err := helpers.BeaconProposerIndex(ctx, head)
	if err != nil {
		return nil, fmt.Errorf("could not calculate proposer index %v", err)
	}
	blk.SetProposerIndex(idx)

	// Set slashings
	proposerSlashings := vs.SlashingsPool.PendingProposerSlashings(ctx, head, false /*noLimit*/)
	validProposerSlashings := make([]*ethpb.ProposerSlashing, 0, len(proposerSlashings))
	for _, slashing := range proposerSlashings {
		_, err := blocks2.ProcessProposerSlashing(ctx, head, slashing, v.SlashValidator)
		if err != nil {
			log.WithError(err).Warn("Proposer: invalid proposer slashing")
			continue
		}
		validProposerSlashings = append(validProposerSlashings, slashing)
	}
	attSlashings := vs.SlashingsPool.PendingAttesterSlashings(ctx, head, false /*noLimit*/)
	validAttSlashings := make([]*ethpb.AttesterSlashing, 0, len(attSlashings))
	for _, slashing := range attSlashings {
		_, err := blocks2.ProcessAttesterSlashing(ctx, head, slashing, v.SlashValidator)
		if err != nil {
			log.WithError(err).Warn("Proposer: invalid attester slashing")
			continue
		}
		validAttSlashings = append(validAttSlashings, slashing)
	}
	blk.Body().SetProposerSlashings(validProposerSlashings)
	blk.Body().SetAttesterSlashings(validAttSlashings)

	// Set exits
	exits := vs.ExitPool.PendingExits(head, req.Slot, false /*noLimit*/)
	validExits := make([]*ethpb.SignedVoluntaryExit, 0, len(exits))
	for _, exit := range exits {
		val, err := head.ValidatorAtIndexReadOnly(exit.Exit.ValidatorIndex)
		if err != nil {
			log.WithError(err).Warn("Proposer: invalid exit")
			continue
		}
		if err := blocks2.VerifyExitAndSignature(val, head.Slot(), head.Fork(), exit, head.GenesisValidatorsRoot()); err != nil {
			log.WithError(err).Warn("Proposer: invalid exit")
			continue
		}
		validExits = append(validExits, exit)
	}
	blk.Body().SetVoluntaryExits(validExits)

	// Set sync aggregate. New in Altair
	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().AltairForkEpoch {
		syncAggregate, err := vs.getSyncAggregate(ctx, req.Slot-1, bytesutil.ToBytes32(parentRoot))
		if err != nil {
			return nil, errors.Wrap(err, "could not compute the sync aggregate")
		}
		if err := blk.Body().SetSyncAggregate(syncAggregate); err != nil {
			return nil, errors.Wrap(err, "could not set sync aggregate")
		}
	}

	// Set execution data. New in Bellatrix
	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().BellatrixForkEpoch {
		fallBackToLocal := true
		canUseBuilder, err := vs.canUseBuilder(ctx, req.Slot, idx)
		if err != nil {
			log.WithError(err).Warn("Proposer: failed to check if builder can be used")
		}
		if canUseBuilder && err != nil {
			h, err := vs.getPayloadHeaderFromBuilder(ctx, req.Slot, idx)
			if err != nil {
				log.WithError(err).Warn("Proposer: failed to get payload header from builder")
			} else {
				blk.SetBlinded(true)
				if err := blk.Body().SetExecution(h); err != nil {
					log.WithError(err).Warn("Proposer: failed to set execution payload")
				} else {
					fallBackToLocal = false
				}
			}
		}
		if fallBackToLocal {
			executionData, err := vs.getExecutionPayload(ctx, req.Slot, idx, bytesutil.ToBytes32(parentRoot), head)
			if err != nil {
				return nil, errors.Wrap(err, "could not get execution payload")
			}
			if err := blk.Body().SetExecution(executionData); err != nil {
				return nil, errors.Wrap(err, "could not set execution payload")
			}
		}
	}

	// Set bls to execution change. New in Capella
	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().CapellaForkEpoch {
		changes, err := vs.BLSChangesPool.BLSToExecChangesForInclusion(head)
		if err != nil {
			return nil, errors.Wrap(err, "could not pack BLSToExecutionChanges")
		}
		if err := blk.Body().SetBLSToExecutionChanges(changes); err != nil {
			return nil, errors.Wrap(err, "could not set BLSToExecutionChanges")
		}
	}

	if err := sBlk.SetBlock(blk); err != nil {
		return nil, err
	}
	sr, err := vs.computeStateRoot(ctx, sBlk)
	if err != nil {
		return nil, fmt.Errorf("could not compute state root: %v", err)
	}
	blk.SetStateRoot(sr)

	pb, err := blk.Proto()
	if err != nil {
		return nil, err
	}
	switch {
	case slots.ToEpoch(req.Slot) < params.BeaconConfig().AltairForkEpoch:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Phase0{Phase0: pb.(*ethpb.BeaconBlock)}}, nil
	case slots.ToEpoch(req.Slot) < params.BeaconConfig().BellatrixForkEpoch:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Altair{Altair: pb.(*ethpb.BeaconBlockAltair)}}, nil
	case slots.ToEpoch(req.Slot) < params.BeaconConfig().CapellaForkEpoch:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Bellatrix{Bellatrix: pb.(*ethpb.BeaconBlockBellatrix)}}, nil
	}
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Capella{Capella: pb.(*ethpb.BeaconBlockCapella)}}, nil
}

// ProposeBeaconBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (vs *Server) ProposeBeaconBlock(ctx context.Context, req *ethpb.GenericSignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.ProposeBeaconBlock")
	defer span.End()
	blk, err := blocks.NewSignedBeaconBlock(req.Block)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not decode block: %v", err)
	}
	return vs.proposeGenericBeaconBlock(ctx, blk)
}

// PrepareBeaconProposer caches and updates the fee recipient for the given proposer.
func (vs *Server) PrepareBeaconProposer(
	ctx context.Context, request *ethpb.PrepareBeaconProposerRequest,
) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "validator.PrepareBeaconProposer")
	defer span.End()
	var feeRecipients []common.Address
	var validatorIndices []types.ValidatorIndex

	newRecipients := make([]*ethpb.PrepareBeaconProposerRequest_FeeRecipientContainer, 0, len(request.Recipients))
	for _, r := range request.Recipients {
		f, err := vs.BeaconDB.FeeRecipientByValidatorID(ctx, r.ValidatorIndex)
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
	if err := vs.BeaconDB.SaveFeeRecipientsByValidatorIDs(ctx, validatorIndices, feeRecipients); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not save fee recipients: %v", err)
	}
	log.WithFields(logrus.Fields{
		"validatorIndices": validatorIndices,
	}).Info("Updated fee recipient addresses for validator indices")
	return &emptypb.Empty{}, nil
}

// GetFeeRecipientByPubKey returns a fee recipient from the beacon node's settings or db based on a given public key
func (vs *Server) GetFeeRecipientByPubKey(ctx context.Context, request *ethpb.FeeRecipientByPubKeyRequest) (*ethpb.FeeRecipientByPubKeyResponse, error) {
	ctx, span := trace.StartSpan(ctx, "validator.GetFeeRecipientByPublicKey")
	defer span.End()
	if request == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request was empty")
	}

	resp, err := vs.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: request.PublicKey})
	if err != nil {
		if strings.Contains(err.Error(), "Could not find validator index") {
			return &ethpb.FeeRecipientByPubKeyResponse{
				FeeRecipient: params.BeaconConfig().DefaultFeeRecipient.Bytes(),
			}, nil
		} else {
			log.WithError(err).Error("An error occurred while retrieving validator index")
			return nil, err
		}

	}
	address, err := vs.BeaconDB.FeeRecipientByValidatorID(ctx, resp.GetIndex())
	if err != nil {
		if errors.Is(err, kv.ErrNotFoundFeeRecipient) {
			return &ethpb.FeeRecipientByPubKeyResponse{
				FeeRecipient: params.BeaconConfig().DefaultFeeRecipient.Bytes(),
			}, nil
		} else {
			log.WithError(err).Error("An error occurred while retrieving fee recipient from db")
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}
	return &ethpb.FeeRecipientByPubKeyResponse{
		FeeRecipient: address.Bytes(),
	}, nil
}

func (vs *Server) proposeGenericBeaconBlock(ctx context.Context, blk interfaces.SignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.proposeGenericBeaconBlock")
	defer span.End()
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("could not tree hash block: %v", err)
	}

	blk, err = vs.unblindBuilderBlock(ctx, blk)
	if err != nil {
		return nil, err
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

	if blk.Version() == version.EIP4844 {
		if err := vs.proposeBlockAndBlobs(ctx, root, blk); err != nil {
			return nil, errors.Wrap(err, "could not propose block and blob")
		}
	} else {
		// Broadcast the new block to the network.
		blkPb, err := blk.Proto()
		if err != nil {
			return nil, errors.Wrap(err, "could not get protobuf block")
		}
		if err := vs.P2P.Broadcast(ctx, blkPb); err != nil {
			return nil, fmt.Errorf("could not broadcast block: %v", err)
		}
		log.WithFields(logrus.Fields{
			"blockRoot": hex.EncodeToString(root[:]),
		}).Debug("Broadcasting block")

		if err := vs.BlockReceiver.ReceiveBlock(ctx, blk, root); err != nil {
			return nil, fmt.Errorf("could not process beacon block: %v", err)
		}

	}

	return &ethpb.ProposeResponse{
		BlockRoot: root[:],
	}, nil
}

func (vs *Server) proposeBlockAndBlobs(ctx context.Context, root [32]byte, blk interfaces.SignedBeaconBlock) error {
	blkPb, err := blk.Pb4844Block()
	if err != nil {
		return errors.Wrap(err, "could not get protobuf block")
	}
	sc, err := vs.BlobsCache.Get(blk.Block().Slot())
	if err != nil {
		return errors.Wrap(err, "could not get blobs from cache")
	}
	if err := vs.P2P.Broadcast(ctx, &ethpb.SignedBeaconBlockAndBlobsSidecar{
		BeaconBlock:  blkPb,
		BlobsSidecar: sc,
	}); err != nil {
		return fmt.Errorf("could not broadcast block: %v", err)
	}
	if err := vs.BlockReceiver.ReceiveBlock(ctx, blk, root); err != nil {
		return fmt.Errorf("could not process beacon block: %v", err)
	}
	if err := vs.BeaconDB.SaveBlobsSidecar(ctx, sc); err != nil {
		return errors.Wrap(err, "could not save sidecar to DB")
	}
	return nil
}

// computeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (vs *Server) computeStateRoot(ctx context.Context, block interfaces.SignedBeaconBlock) ([]byte, error) {
	beaconState, err := vs.StateGen.StateByRoot(ctx, block.Block().ParentRoot())
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

// SubmitValidatorRegistrations submits validator registrations.
func (vs *Server) SubmitValidatorRegistrations(ctx context.Context, reg *ethpb.SignedValidatorRegistrationsV1) (*emptypb.Empty, error) {
	if vs.BlockBuilder == nil || !vs.BlockBuilder.Configured() {
		return &emptypb.Empty{}, status.Errorf(codes.InvalidArgument, "Could not register block builder: %v", builder.ErrNoBuilder)
	}

	if err := vs.BlockBuilder.RegisterValidator(ctx, reg.Messages); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not register block builder: %v", err)
	}

	return &emptypb.Empty{}, nil
}
