package blockchain

import (
	"bytes"
	"context"
	"encoding/hex"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// BlockReceiver interface defines the methods of chain service receive and processing new blocks.
type BlockReceiver interface {
	ReceiveBlock(ctx context.Context, block *ethpb.SignedBeaconBlock) error
	ReceiveBlockNoPubsub(ctx context.Context, block *ethpb.SignedBeaconBlock) error
	ReceiveBlockNoPubsubForkchoice(ctx context.Context, block *ethpb.SignedBeaconBlock) error
	ReceiveBlockNoVerify(ctx context.Context, block *ethpb.SignedBeaconBlock) error
	HasInitSyncBlock(root [32]byte) bool
}

// ReceiveBlock is a function that defines the operations that are preformed on
// blocks that is received from rpc service. The operations consists of:
//   1. Gossip block to other peers
//   2. Validate block, apply state transition and update check points
//   3. Apply fork choice to the processed block
//   4. Save latest head info
func (s *Service) ReceiveBlock(ctx context.Context, block *ethpb.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlock")
	defer span.End()

	root, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root on received block")
	}

	// Broadcast the new block to the network.
	if err := s.p2p.Broadcast(ctx, block); err != nil {
		return errors.Wrap(err, "could not broadcast block")
	}
	log.WithFields(logrus.Fields{
		"blockRoot": hex.EncodeToString(root[:]),
	}).Debug("Broadcasting block")

	if err := captureSentTimeMetric(uint64(s.genesisTime.Unix()), block.Block.Slot); err != nil {
		// If a node fails to capture metric, this shouldn't cause the block processing to fail.
		log.Warnf("Could not capture block sent time metric: %v", err)
	}

	if err := s.ReceiveBlockNoPubsub(ctx, block); err != nil {
		return err
	}

	return nil
}

// ReceiveBlockNoPubsub is a function that defines the the operations (minus pubsub)
// that are preformed on blocks that is received from regular sync service. The operations consists of:
//   1. Validate block, apply state transition and update check points
//   2. Apply fork choice to the processed block
//   3. Save latest head info
func (s *Service) ReceiveBlockNoPubsub(ctx context.Context, block *ethpb.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlockNoPubsub")
	defer span.End()
	blockCopy := stateTrie.CopySignedBeaconBlock(block)

	// Apply state transition on the new block.
	postState, err := s.onBlock(ctx, blockCopy)
	if err != nil {
		err := errors.Wrap(err, "could not process block")
		traceutil.AnnotateError(span, err)
		return err
	}

	// Add attestations from the block to the pool for fork choice.
	if err := s.attPool.SaveBlockAttestations(blockCopy.Block.Body.Attestations); err != nil {
		log.Errorf("Could not save attestation for fork choice: %v", err)
		return nil
	}
	for _, exit := range block.Block.Body.VoluntaryExits {
		s.exitPool.MarkIncluded(exit)
	}

	s.epochParticipationLock.Lock()
	defer s.epochParticipationLock.Unlock()
	s.epochParticipation[helpers.SlotToEpoch(blockCopy.Block.Slot)] = precompute.Balances

	root, err := stateutil.BlockRoot(blockCopy.Block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root on received block")
	}

	if featureconfig.Get().DisableForkChoice && block.Block.Slot > s.headSlot() {
		if err := s.saveHead(ctx, root); err != nil {
			return errors.Wrap(err, "could not save head")
		}
	} else {
		if err := s.updateHead(ctx, postState.Balances()); err != nil {
			return errors.Wrap(err, "could not save head")
		}
	}

	// Send notification of the processed block to the state feed.
	s.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			Slot:      blockCopy.Block.Slot,
			BlockRoot: root,
			Verified:  true,
		},
	})

	// Reports on block and fork choice metrics.
	reportSlotMetrics(blockCopy.Block.Slot, s.headSlot(), s.CurrentSlot(), s.finalizedCheckpt)

	// Log block sync status.
	logBlockSyncStatus(blockCopy.Block, root, s.finalizedCheckpt)

	// Log state transition data.
	logStateTransitionData(blockCopy.Block)

	return nil
}

// ReceiveBlockNoPubsubForkchoice is a function that defines the all operations (minus pubsub and forkchoice)
// that are preformed blocks that is received from initial sync service. The operations consists of:
//   1. Validate block, apply state transition and update check points
//   2. Save latest head info
func (s *Service) ReceiveBlockNoPubsubForkchoice(ctx context.Context, block *ethpb.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlockNoForkchoice")
	defer span.End()
	blockCopy := stateTrie.CopySignedBeaconBlock(block)

	// Apply state transition on the new block.
	_, err := s.onBlock(ctx, blockCopy)
	if err != nil {
		err := errors.Wrap(err, "could not process block")
		traceutil.AnnotateError(span, err)
		return err
	}

	root, err := stateutil.BlockRoot(blockCopy.Block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root on received block")
	}
	cachedHeadRoot, err := s.HeadRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head root from cache")
	}
	if !bytes.Equal(root[:], cachedHeadRoot) {
		if err := s.saveHead(ctx, root); err != nil {
			return errors.Wrap(err, "could not save head")
		}
	}

	// Send notification of the processed block to the state feed.
	s.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			Slot:      blockCopy.Block.Slot,
			BlockRoot: root,
			Verified:  true,
		},
	})

	// Reports on block and fork choice metrics.
	reportSlotMetrics(blockCopy.Block.Slot, s.headSlot(), s.CurrentSlot(), s.finalizedCheckpt)

	// Log block sync status.
	logBlockSyncStatus(blockCopy.Block, root, s.finalizedCheckpt)

	// Log state transition data.
	logStateTransitionData(blockCopy.Block)

	s.epochParticipationLock.Lock()
	defer s.epochParticipationLock.Unlock()
	s.epochParticipation[helpers.SlotToEpoch(blockCopy.Block.Slot)] = precompute.Balances

	return nil
}

// ReceiveBlockNoVerify runs state transition on a input block without verifying the block's BLS contents.
// Depends on the security model, this is the "minimal" work a node can do to sync the chain.
// It simulates light client behavior and assumes 100% trust with the syncing peer.
func (s *Service) ReceiveBlockNoVerify(ctx context.Context, block *ethpb.SignedBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlockNoVerify")
	defer span.End()
	blockCopy := stateTrie.CopySignedBeaconBlock(block)

	// Apply state transition on the incoming newly received blockCopy without verifying its BLS contents.
	if err := s.onBlockInitialSyncStateTransition(ctx, blockCopy); err != nil {
		err := errors.Wrap(err, "could not process block")
		traceutil.AnnotateError(span, err)
		return err
	}

	root, err := stateutil.BlockRoot(blockCopy.Block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root on received blockCopy")
	}

	cachedHeadRoot, err := s.HeadRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head root from cache")
	}

	if !bytes.Equal(root[:], cachedHeadRoot) {
		if err := s.saveHeadNoDB(ctx, blockCopy, root); err != nil {
			err := errors.Wrap(err, "could not save head")
			traceutil.AnnotateError(span, err)
			return err
		}
	}

	// Send notification of the processed block to the state feed.
	s.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			Slot:      blockCopy.Block.Slot,
			BlockRoot: root,
			Verified:  false,
		},
	})

	// Reports on blockCopy and fork choice metrics.
	reportSlotMetrics(blockCopy.Block.Slot, s.headSlot(), s.CurrentSlot(), s.finalizedCheckpt)

	// Log state transition data.
	log.WithFields(logrus.Fields{
		"slot":         blockCopy.Block.Slot,
		"attestations": len(blockCopy.Block.Body.Attestations),
		"deposits":     len(blockCopy.Block.Body.Deposits),
	}).Debug("Finished applying state transition")

	s.epochParticipationLock.Lock()
	defer s.epochParticipationLock.Unlock()
	s.epochParticipation[helpers.SlotToEpoch(blockCopy.Block.Slot)] = precompute.Balances

	return nil
}

// HasInitSyncBlock returns true if the block of the input root exists in initial sync blocks cache.
func (s *Service) HasInitSyncBlock(root [32]byte) bool {
	return s.hasInitSyncBlock(root)
}
