package core

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// eth1DataNotification is a latch to stop flooding logs with the same warning.
var eth1DataNotification bool

const (
	// CouldNotDecodeBlock means that a signed beacon block couldn't be created from the block present in the request.
	couldNotDecodeBlock = "could not decode block"
	eth1dataTimeout     = 2 * time.Second
)

// GetBeaconBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot.
func (s *Service) GetBeaconBlock(ctx context.Context, slot primitives.Slot, randaoReveal []byte, graffiti []byte) (*ethpb.GenericBeaconBlock, *RpcError) {
	ctx, span := trace.StartSpan(ctx, "proposer.GetBeaconBlock")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(slot))) // lint:ignore uintcast -- This is OK for tracing.

	t, err := slots.ToTime(uint64(s.TimeFetcher.GenesisTime().Unix()), slot)
	if err != nil {
		log.WithError(err).Error("Could not convert slot to time")
	}
	log.WithFields(logrus.Fields{
		"slot":               slot,
		"sinceSlotStartTime": time.Since(t),
	}).Info("Begin building block")

	// A syncing validator should not produce a block.
	if s.SyncChecker.Syncing() {
		return nil, &RpcError{Reason: Unavailable, Err: errors.New("syncing to latest head, not ready to respond")}
	}

	// process attestations and update head in forkchoice
	s.ForkchoiceFetcher.UpdateHead(ctx, s.TimeFetcher.CurrentSlot())
	headRoot := s.ForkchoiceFetcher.CachedHeadRoot()
	parentRoot := s.ForkchoiceFetcher.GetProposerHead()
	if parentRoot != headRoot {
		blockchain.LateBlockAttemptedReorgCount.Inc()
	}

	// An optimistic validator MUST NOT produce a block (i.e., sign across the DOMAIN_BEACON_PROPOSER domain).
	if slots.ToEpoch(slot) >= params.BeaconConfig().BellatrixForkEpoch {
		if err := s.optimisticStatus(ctx); err != nil {
			return nil, &RpcError{Reason: Unavailable, Err: errors.Wrapf(err, "validator is not ready to propose")}
		}
	}

	sBlk, err := getEmptyBlock(slot)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not prepare block")}
	}
	head, err := s.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not get head state")}
	}
	head, err = transition.ProcessSlotsUsingNextSlotCache(ctx, head, parentRoot[:], slot)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, fmt.Sprintf("could not process slots up to %d", slot))}
	}

	// Set slot, graffiti, randao reveal, and parent root.
	sBlk.SetSlot(slot)
	sBlk.SetGraffiti(graffiti)
	sBlk.SetRandaoReveal(randaoReveal)
	sBlk.SetParentRoot(parentRoot[:])

	// Set proposer index.
	idx, err := helpers.BeaconProposerIndex(ctx, head)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not calculate proposer index")}
	}
	sBlk.SetProposerIndex(idx)

	var blobBundle *enginev1.BlobsBundle
	var blindBlobBundle *enginev1.BlindedBlobsBundle
	if features.Get().BuildBlockParallel {
		blindBlobBundle, blobBundle, err = s.buildBlockParallel(ctx, sBlk, head)
		if err != nil {
			return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not build block in parallel")}
		}
	} else {
		// Set eth1 data.
		eth1Data, err := s.eth1DataMajorityVote(ctx, head)
		if err != nil {
			eth1Data = &ethpb.Eth1Data{DepositRoot: params.BeaconConfig().ZeroHash[:], BlockHash: params.BeaconConfig().ZeroHash[:]}
			log.WithError(err).Error("Could not get eth1data")
		}
		sBlk.SetEth1Data(eth1Data)

		// Set deposit and attestation.
		deposits, atts, err := s.packDepositsAndAttestations(ctx, head, eth1Data) // TODO: split attestations and deposits
		if err != nil {
			sBlk.SetDeposits([]*ethpb.Deposit{})
			sBlk.SetAttestations([]*ethpb.Attestation{})
			log.WithError(err).Error("Could not pack deposits and attestations")
		} else {
			sBlk.SetDeposits(deposits)
			sBlk.SetAttestations(atts)
		}

		// Set slashings.
		validProposerSlashings, validAttSlashings := s.getSlashings(ctx, head)
		sBlk.SetProposerSlashings(validProposerSlashings)
		sBlk.SetAttesterSlashings(validAttSlashings)

		// Set exits.
		sBlk.SetVoluntaryExits(s.getExits(head, slot))

		// Set sync aggregate. New in Altair.
		s.setSyncAggregate(ctx, sBlk)

		// Get local and builder (if enabled) payloads. Set execution data. New in Bellatrix.
		var overrideBuilder bool
		var localPayload interfaces.ExecutionData
		localPayload, blobBundle, overrideBuilder, err = s.getLocalPayloadAndBlobs(ctx, sBlk.Block(), head)
		if err != nil {
			return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not get local payload")}
		}
		// There's no reason to try to get a builder bid if local override is true.
		var builderPayload interfaces.ExecutionData
		if !overrideBuilder {
			builderPayload, blindBlobBundle, err = s.getBuilderPayloadAndBlobs(ctx, sBlk.Block().Slot(), sBlk.Block().ProposerIndex())
			if err != nil {
				builderGetPayloadMissCount.Inc()
				log.WithError(err).Error("Could not get builder payload")
			}
		}
		if err := setExecutionData(ctx, sBlk, localPayload, builderPayload); err != nil {
			return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not set execution data")}
		}

		// Set bls to execution change. New in Capella.
		s.setBlsToExecData(sBlk, head)

		if err := setKzgCommitments(sBlk, blobBundle, blindBlobBundle); err != nil {
			return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not set kzg commitment")}
		}
	}

	sr, err := s.computeStateRoot(ctx, sBlk)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not compute state root")}
	}
	sBlk.SetStateRoot(sr)

	fullBlobs, err := blobsBundleToSidecars(blobBundle, sBlk.Block())
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not convert blobs bundle to sidecar")}
	}
	blindBlobs, err := blindBlobsBundleToSidecars(blindBlobBundle, sBlk.Block())
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not convert blind blobs bundle to sidecar")}
	}

	log.WithFields(logrus.Fields{
		"slot":               slot,
		"sinceSlotStartTime": time.Since(t),
		"validator":          sBlk.Block().ProposerIndex(),
	}).Info("Finished building block")

	pb, err := sBlk.Block().Proto()
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not convert block to proto")}
	}
	if slots.ToEpoch(slot) >= params.BeaconConfig().DenebForkEpoch {
		if sBlk.IsBlinded() {
			blockAndBlobs := &ethpb.BlindedBeaconBlockAndBlobsDeneb{
				Block: pb.(*ethpb.BlindedBeaconBlockDeneb),
				Blobs: blindBlobs,
			}
			return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedDeneb{BlindedDeneb: blockAndBlobs}}, nil
		}

		blockAndBlobs := &ethpb.BeaconBlockAndBlobsDeneb{
			Block: pb.(*ethpb.BeaconBlockDeneb),
			Blobs: fullBlobs,
		}
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Deneb{Deneb: blockAndBlobs}}, nil
	}

	if slots.ToEpoch(slot) >= params.BeaconConfig().CapellaForkEpoch {
		if sBlk.IsBlinded() {
			return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedCapella{BlindedCapella: pb.(*ethpb.BlindedBeaconBlockCapella)}, IsBlinded: true, PayloadValue: sBlk.ValueInGwei()}, nil
		}
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Capella{Capella: pb.(*ethpb.BeaconBlockCapella)}, IsBlinded: false, PayloadValue: sBlk.ValueInGwei()}, nil
	}
	if slots.ToEpoch(slot) >= params.BeaconConfig().BellatrixForkEpoch {
		if sBlk.IsBlinded() {
			return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: pb.(*ethpb.BlindedBeaconBlockBellatrix)}, IsBlinded: true, PayloadValue: sBlk.ValueInGwei()}, nil
		}
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Bellatrix{Bellatrix: pb.(*ethpb.BeaconBlockBellatrix)}, IsBlinded: false, PayloadValue: sBlk.ValueInGwei()}, nil
	}
	if slots.ToEpoch(slot) >= params.BeaconConfig().AltairForkEpoch {
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Altair{Altair: pb.(*ethpb.BeaconBlockAltair)}, IsBlinded: false, PayloadValue: 0}, nil
	}
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Phase0{Phase0: pb.(*ethpb.BeaconBlock)}, IsBlinded: false, PayloadValue: 0}, nil
}

// ProposeBeaconBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (s *Service) ProposeBeaconBlock(ctx context.Context, req *ethpb.GenericSignedBeaconBlock) ([]byte, *RpcError) {
	ctx, span := trace.StartSpan(ctx, "proposer.ProposeBeaconBlock")
	defer span.End()

	blk, err := blocks.NewSignedBeaconBlock(req.Block)
	if err != nil {
		return nil, &RpcError{Reason: BadRequest, Err: errors.Wrap(err, couldNotDecodeBlock)}
	}

	var blindSidecars []*ethpb.SignedBlindedBlobSidecar
	if blk.Version() >= version.Deneb && blk.IsBlinded() {
		blindSidecars = req.GetBlindedDeneb().SignedBlindedBlobSidecars
	}

	unblinder, err := newUnblinder(blk, blindSidecars, s.BlockBuilder)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not create unblinder")}
	}
	blinded := unblinder.b.IsBlinded()

	blk, unblindedSidecars, err := unblinder.unblindBuilderBlock(ctx)
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not unblind builder block")}
	}

	// Broadcast the new block to the network.
	blkPb, err := blk.Proto()
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not get protobuf block")}
	}
	if err := s.P2P.Broadcast(ctx, blkPb); err != nil {
		return nil, &RpcError{Reason: Internal, Err: fmt.Errorf("could not broadcast block: %v", err)}
	}

	var scs []*ethpb.SignedBlobSidecar
	if blk.Version() >= version.Deneb {
		if blinded {
			scs = unblindedSidecars // Use sidecars from unblinder if the block was blinded.
		} else {
			scs, err = extraSidecars(req) // Use sidecars from the request if the block was not blinded.
			if err != nil {
				return nil, &RpcError{Reason: Internal, Err: errors.Wrap(err, "could not extract blobs")}
			}
		}
		sidecars := make([]*ethpb.BlobSidecar, len(scs))
		for i, sc := range scs {
			log.WithFields(logrus.Fields{
				"blockRoot": hex.EncodeToString(sc.Message.BlockRoot),
				"index":     sc.Message.Index,
			}).Debug("Broadcasting blob sidecar")
			if err := s.P2P.BroadcastBlob(ctx, sc.Message.Index, sc); err != nil {
				log.WithError(err).Errorf("Could not broadcast blob sidecar index %d / %d", i, len(scs))
			}
			sidecars[i] = sc.Message
		}
		if len(scs) > 0 {
			if err := s.BeaconDB.SaveBlobSidecar(ctx, sidecars); err != nil {
				return nil, &RpcError{Reason: Internal, Err: err}
			}
		}
	}

	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, &RpcError{Reason: Internal, Err: fmt.Errorf("could not tree hash block: %v", err)}
	}
	log.WithFields(logrus.Fields{
		"blockRoot": hex.EncodeToString(root[:]),
	}).Debug("Broadcasting block")

	if err := s.BlockReceiver.ReceiveBlock(ctx, blk, root); err != nil {
		return nil, &RpcError{Reason: Internal, Err: fmt.Errorf("could not process beacon block: %v", err)}
	}

	log.WithField("slot", blk.Block().Slot()).Debugf(
		"Block proposal received via RPC")
	s.BlockNotifier.BlockFeed().Send(&feed.Event{
		Type: blockfeed.ReceivedBlock,
		Data: &blockfeed.ReceivedBlockData{SignedBlock: blk},
	})

	return root[:], nil
}

// optimisticStatus returns an error if the node is currently optimistic with respect to head.
// by definition, an optimistic node is not a full node. It is unable to produce blocks,
// since an execution engine cannot produce a payload upon an unknown parent.
// It cannot faithfully attest to the head block of the chain, since it has not fully verified that block.
//
// Spec:
// https://github.com/ethereum/consensus-specs/blob/dev/sync/optimistic.md
func (s *Service) optimisticStatus(ctx context.Context) error {
	if slots.ToEpoch(s.TimeFetcher.CurrentSlot()) < params.BeaconConfig().BellatrixForkEpoch {
		return nil
	}
	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		return errors.Wrap(err, "could not determine if the node is a optimistic node")
	}
	if !optimistic {
		return nil
	}

	return errOptimisticMode
}

func (s *Service) buildBlockParallel(
	ctx context.Context,
	sBlk interfaces.SignedBeaconBlock,
	head state.BeaconState,
) (*enginev1.BlindedBlobsBundle, *enginev1.BlobsBundle, error) {
	// Build consensus fields in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Set eth1 data.
		eth1Data, err := s.eth1DataMajorityVote(ctx, head)
		if err != nil {
			eth1Data = &ethpb.Eth1Data{DepositRoot: params.BeaconConfig().ZeroHash[:], BlockHash: params.BeaconConfig().ZeroHash[:]}
			log.WithError(err).Error("Could not get eth1data")
		}
		sBlk.SetEth1Data(eth1Data)

		// Set deposit and attestation.
		deposits, atts, err := s.packDepositsAndAttestations(ctx, head, eth1Data) // TODO: split attestations and deposits
		if err != nil {
			sBlk.SetDeposits([]*ethpb.Deposit{})
			sBlk.SetAttestations([]*ethpb.Attestation{})
			log.WithError(err).Error("Could not pack deposits and attestations")
		} else {
			sBlk.SetDeposits(deposits)
			sBlk.SetAttestations(atts)
		}

		// Set slashings.
		validProposerSlashings, validAttSlashings := s.getSlashings(ctx, head)
		sBlk.SetProposerSlashings(validProposerSlashings)
		sBlk.SetAttesterSlashings(validAttSlashings)

		// Set exits.
		sBlk.SetVoluntaryExits(s.getExits(head, sBlk.Block().Slot()))

		// Set sync aggregate. New in Altair.
		s.setSyncAggregate(ctx, sBlk)

		// Set bls to execution change. New in Capella.
		s.setBlsToExecData(sBlk, head)
	}()

	localPayload, blobsBundle, overrideBuilder, err := s.getLocalPayloadAndBlobs(ctx, sBlk.Block(), head)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get local payload")
	}

	// There's no reason to try to get a builder bid if local override is true.
	var builderPayload interfaces.ExecutionData
	var blindBlobsBundle *enginev1.BlindedBlobsBundle
	if !overrideBuilder {
		builderPayload, blindBlobsBundle, err = s.getBuilderPayloadAndBlobs(ctx, sBlk.Block().Slot(), sBlk.Block().ProposerIndex())
		if err != nil {
			builderGetPayloadMissCount.Inc()
			log.WithError(err).Error("Could not get builder payload")
		}
	}

	if err := setExecutionData(ctx, sBlk, localPayload, builderPayload); err != nil {
		return nil, nil, errors.Wrap(err, "could not set execution data")
	}

	if err := setKzgCommitments(sBlk, blobsBundle, blindBlobsBundle); err != nil {
		return nil, nil, errors.Wrap(err, "could not set kzg commitments")
	}

	wg.Wait() // Wait until block is built via consensus and execution fields.

	return blindBlobsBundle, blobsBundle, nil
}

// computeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (s *Service) computeStateRoot(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	beaconState, err := s.StateGen.StateByRoot(ctx, block.Block().ParentRoot())
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

// extraSidecars extracts the sidecars from the request.
// return error if there are too many sidecars.
func extraSidecars(req *ethpb.GenericSignedBeaconBlock) ([]*ethpb.SignedBlobSidecar, error) {
	b, ok := req.GetBlock().(*ethpb.GenericSignedBeaconBlock_Deneb)
	if !ok {
		return nil, errors.New("Could not cast block to Deneb")
	}
	if len(b.Deneb.Blobs) > fieldparams.MaxBlobsPerBlock {
		return nil, fmt.Errorf("too many blobs in block: %d", len(b.Deneb.Blobs))
	}
	return b.Deneb.Blobs, nil
}
