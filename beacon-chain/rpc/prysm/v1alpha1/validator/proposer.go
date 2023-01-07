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
	"github.com/protolambda/go-kzg/eth"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/builder"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/kv"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blobs"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
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
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	// An optimistic validator MUST NOT produce a block (i.e., sign across the DOMAIN_BEACON_PROPOSER domain).
	if err := vs.optimisticStatus(ctx); err != nil {
		return nil, status.Errorf(codes.Unavailable, "Validator is not ready to propose: %v", err)
	}

	sBlk, err := emptyBlockToSign(req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not prepare block: %v", err)
	}

	parentRoot, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head root: %v", err)
	}
	head, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	head, err = transition.ProcessSlotsUsingNextSlotCache(ctx, head, parentRoot, req.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", req.Slot, err)
	}

	blk := sBlk.Block()
	// Set slot, graffiti, randao reveal, and parent root.
	blk.SetSlot(req.Slot)
	blk.Body().SetGraffiti(req.Graffiti)
	blk.Body().SetRandaoReveal(req.RandaoReveal)
	blk.SetParentRoot(parentRoot)

	// Set eth1 data.
	eth1Data, err := vs.eth1DataMajorityVote(ctx, head)
	if err != nil {
		log.WithError(err).Error("Could not get eth1data")
	} else {
		blk.Body().SetEth1Data(eth1Data)

		// Set deposit and attestation.
		deposits, atts, err := vs.packDepositsAndAttestations(ctx, head, eth1Data) // TODO: split attestations and deposits
		if err != nil {
			log.WithError(err).Error("Could not pack deposits and attestations")
		} else {
			blk.Body().SetDeposits(deposits)
			blk.Body().SetAttestations(atts)
		}
	}

	// Set proposer index
	idx, err := helpers.BeaconProposerIndex(ctx, head)
	if err != nil {
		return nil, fmt.Errorf("could not calculate proposer index %v", err)
	}
	blk.SetProposerIndex(idx)

	// Set slashings
	validProposerSlashings, validAttSlashings := vs.getSlashings(ctx, head)
	blk.Body().SetProposerSlashings(validProposerSlashings)
	blk.Body().SetAttesterSlashings(validAttSlashings)

	// Set exits
	blk.Body().SetVoluntaryExits(vs.getExits(head, req.Slot))

	// Set sync aggregate. New in Altair.
	if req.Slot > 0 && slots.ToEpoch(req.Slot) >= params.BeaconConfig().AltairForkEpoch {
		syncAggregate, err := vs.getSyncAggregate(ctx, req.Slot-1, bytesutil.ToBytes32(parentRoot))
		if err != nil {
			log.WithError(err).Error("Could not get sync aggregate")
		} else {
			if err := blk.Body().SetSyncAggregate(syncAggregate); err != nil {
				log.WithError(err).Error("Could not set sync aggregate")
				if err := blk.Body().SetSyncAggregate(&ethpb.SyncAggregate{
					SyncCommitteeBits:      make([]byte, params.BeaconConfig().SyncCommitteeSize),
					SyncCommitteeSignature: make([]byte, fieldparams.BLSSignatureLength),
				}); err != nil {
					return nil, status.Errorf(codes.Internal, "Could not set default sync aggregate: %v", err)
				}
			}
		}
	}

	// Set execution data. New in Bellatrix
	var bundle *enginev1.BlobsBundle
	var executionData interfaces.ExecutionData
	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().BellatrixForkEpoch {
		fallBackToLocal := true
		canUseBuilder, err := vs.canUseBuilder(ctx, req.Slot, idx)
		if err != nil {
			log.WithError(err).Warn("Proposer: failed to check if builder can be used")
		} else if canUseBuilder {
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
			switch {
			case slots.ToEpoch(req.Slot) < params.BeaconConfig().EIP4844ForkEpoch:
				executionData, err := vs.getExecutionPayload(ctx, req.Slot, idx, bytesutil.ToBytes32(parentRoot), head)
				if err != nil {
					return nil, errors.Wrap(err, "could not get execution payload")
				}
				if err := blk.Body().SetExecution(executionData); err != nil {
					return nil, errors.Wrap(err, "could not set execution payload")
				}
			default:
				executionData, bundle, err = vs.getExecutionPayloadV2AndBlobsBundleV1(ctx, req.Slot, idx, bytesutil.ToBytes32(parentRoot), head)
				if err != nil {
					return nil, errors.Wrap(err, "could not get execution payload")
				}
				if err := blk.Body().SetExecution(executionData); err != nil {
					return nil, errors.Wrap(err, "could not set execution payload")
				}
				if err := blk.Body().SetBlobKzgCommitments(bundle.KzgCommitments); err != nil {
					return nil, errors.Wrap(err, "could not set blob kzg commitments")
				}
			}

		}
	}

	// Set bls to execution change. New in Capella
	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().CapellaForkEpoch {
		changes, err := vs.BLSChangesPool.BLSToExecChangesForInclusion(head)
		if err != nil {
			log.WithError(err).Error("Could not get bls to execution changes")
		} else {
			if err := blk.Body().SetBLSToExecutionChanges(changes); err != nil {
				log.WithError(err).Error("Could not set bls to execution changes")
			}
		}
	}

	sr, err := vs.computeStateRoot(ctx, sBlk)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute state root: %v", err)
	}
	blk.SetStateRoot(sr)

	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().EIP4844ForkEpoch {
		aggregatedProof, err := eth.ComputeAggregateKZGProof(blobs.BlobsSequenceImpl(bundle.Blobs))
		if err != nil {
			return nil, fmt.Errorf("failed to compute aggregated kzg proof: %v", err)
		}
		r, err := blk.HashTreeRoot()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not tree hash final block: %v", err)
		}
		vs.BlobsCache.Put(&ethpb.BlobsSidecar{
			BeaconBlockRoot: r[:],
			BeaconBlockSlot: req.Slot,
			Blobs:           bundle.Blobs,
			AggregatedProof: aggregatedProof[:],
		})
	}

	pb, err := blk.Proto()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert block to proto: %v", err)
	}
	if slots.ToEpoch(req.Slot) >= params.BeaconConfig().EIP4844ForkEpoch {
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_EIP4844{EIP4844: pb.(*ethpb.BeaconBlock4844)}}, nil
	} else if slots.ToEpoch(req.Slot) >= params.BeaconConfig().CapellaForkEpoch {
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Capella{Capella: pb.(*ethpb.BeaconBlockCapella)}}, nil
	} else if slots.ToEpoch(req.Slot) >= params.BeaconConfig().BellatrixForkEpoch {
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Bellatrix{Bellatrix: pb.(*ethpb.BeaconBlockBellatrix)}}, nil
	} else if slots.ToEpoch(req.Slot) >= params.BeaconConfig().AltairForkEpoch {
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Altair{Altair: pb.(*ethpb.BeaconBlockAltair)}}, nil
	}
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Phase0{Phase0: pb.(*ethpb.BeaconBlock)}}, nil
}

func emptyBlockToSign(slot types.Slot) (interfaces.SignedBeaconBlock, error) {
	var sBlk interfaces.SignedBeaconBlock
	var err error
	switch {
	case slots.ToEpoch(slot) < params.BeaconConfig().AltairForkEpoch:
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
	case slots.ToEpoch(slot) < params.BeaconConfig().BellatrixForkEpoch:
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{Body: &ethpb.BeaconBlockBodyAltair{}}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
	case slots.ToEpoch(slot) < params.BeaconConfig().CapellaForkEpoch:
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{Block: &ethpb.BeaconBlockBellatrix{Body: &ethpb.BeaconBlockBodyBellatrix{}}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
	case slots.ToEpoch(slot) < params.BeaconConfig().EIP4844ForkEpoch:
		blk, err = blocks.NewBeaconBlock(&ethpb.BeaconBlockCapella{Body: &ethpb.BeaconBlockBodyCapella{}})
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockCapella{Block: &ethpb.BeaconBlockCapella{Body: &ethpb.BeaconBlockBodyCapella{}}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
	default:
		blk, err = blocks.NewBeaconBlock(&ethpb.BeaconBlock4844{Body: &ethpb.BeaconBlockBody4844{}})
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock4844{Block: &ethpb.BeaconBlock4844{Body: &ethpb.BeaconBlockBody4844{}}})
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "Could not initialize block for proposal: %v", err)
		}
	}
	return sBlk, err
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
