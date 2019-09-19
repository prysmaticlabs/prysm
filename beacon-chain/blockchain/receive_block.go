package blockchain

import (
	"bytes"
	"context"
	"encoding/hex"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// BlockReceiver interface defines the methods of chain service receive and processing new blocks.
type BlockReceiver interface {
	ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error
	ReceiveBlockNoPubsub(ctx context.Context, block *ethpb.BeaconBlock) error
	ReceiveBlockNoPubsubForkchoice(ctx context.Context, block *ethpb.BeaconBlock) error
}

// ReceiveBlock is a function that defines the operations that are preformed on
// blocks that is received from rpc service. The operations consists of:
//   1. Gossip block to other peers
//   2. Validate block, apply state transition and update check points
//   3. Apply fork choice to the processed block
//   4. Save latest head info
func (s *Service) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlock")
	defer span.End()

	root, err := ssz.SigningRoot(block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root on received block")
	}

	// Broadcast the new block to the network.
	if err := s.p2p.Broadcast(ctx, block); err != nil {
		return errors.Wrap(err, "could not broadcast block")
	}
	log.WithFields(logrus.Fields{
		"blockRoot": hex.EncodeToString(root[:]),
	}).Info("Broadcasting block")

	if err := s.ReceiveBlockNoPubsub(ctx, block); err != nil {
		return err
	}

	processedBlk.Inc()
	return nil
}

// ReceiveBlockNoPubsub is a function that defines the the operations (minus pubsub)
// that are preformed on blocks that is received from regular sync service. The operations consists of:
//   1. Validate block, apply state transition and update check points
//   2. Apply fork choice to the processed block
//   3. Save latest head info
func (s *Service) ReceiveBlockNoPubsub(ctx context.Context, block *ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlockNoPubsub")
	defer span.End()

	// Apply state transition on the new block.
	if err := s.forkChoiceStore.OnBlock(ctx, block); err != nil {
		return errors.Wrap(err, "could not process block from fork choice service")
	}
	root, err := ssz.SigningRoot(block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root on received block")
	}
	logStateTransitionData(block, root[:])

	// Run fork choice after applying state transition on the new block.
	headRoot, err := s.forkChoiceStore.Head(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head from fork choice service")
	}
	headBlk, err := s.beaconDB.Block(ctx, bytesutil.ToBytes32(headRoot))
	if err != nil {
		return errors.Wrap(err, "could not compute state from block head")
	}
	log.WithFields(logrus.Fields{
		"headSlot": headBlk.Slot,
		"headRoot": hex.EncodeToString(headRoot),
	}).Info("Finished applying fork choice for block")

	isCompetingBlock(root[:], block.Slot, headRoot, headBlk.Slot)

	// Save head info after running fork choice.
	if err := s.saveHead(ctx, headBlk, bytesutil.ToBytes32(headRoot)); err != nil {
		return errors.Wrap(err, "could not save head")
	}

	// Remove block's contained deposits, attestations, and other operations from persistent storage.
	if err := s.cleanupBlockOperations(ctx, block); err != nil {
		return errors.Wrap(err, "could not clean up block deposits, attestations, and other operations")
	}

	// Reports on block and fork choice metrics.
	s.reportSlotMetrics(block.Slot)

	processedBlkNoPubsub.Inc()

	// We write the latest saved head root to a feed for consumption by other services.
	s.headUpdatedFeed.Send(bytesutil.ToBytes32(headRoot))
	return nil
}

// ReceiveBlockNoPubsubForkchoice is a function that defines the all operations (minus pubsub and forkchoice)
// that are preformed blocks that is received from initial sync service. The operations consists of:
//   1. Validate block, apply state transition and update check points
//   2. Save latest head info
func (s *Service) ReceiveBlockNoPubsubForkchoice(ctx context.Context, block *ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlockNoForkchoice")
	defer span.End()

	// Apply state transition on the incoming newly received block.
	if err := s.forkChoiceStore.OnBlock(ctx, block); err != nil {
		return errors.Wrap(err, "could not process block from fork choice service")
	}
	root, err := ssz.SigningRoot(block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root on received block")
	}
	logStateTransitionData(block, root[:])

	// Save new block as head.
	if err := s.saveHead(ctx, block, root); err != nil {
		return errors.Wrap(err, "could not save head")
	}

	// Remove block's contained deposits, attestations, and other operations from persistent storage.
	if err := s.cleanupBlockOperations(ctx, block); err != nil {
		return errors.Wrap(err, "could not clean up block deposits, attestations, and other operations")
	}

	// Reports on block and fork choice metrics.
	s.reportSlotMetrics(block.Slot)

	// We write the latest saved head root to a feed for consumption by other services.
	s.headUpdatedFeed.Send(root)
	processedBlkNoPubsubForkchoice.Inc()
	return nil
}

// cleanupBlockOperations processes and cleans up any block operations relevant to the beacon node
// such as attestations, exits, and deposits. We update the latest seen attestation by validator
// in the local node's runtime, cleanup and remove pending deposits which have been included in the block
// from our node's local cache, and process validator exits and more.
func (s *Service) cleanupBlockOperations(ctx context.Context, block *ethpb.BeaconBlock) error {
	// Forward processed block to operation pool to remove individual operation from DB.
	if s.opsPoolService.IncomingProcessedBlockFeed().Send(block) == 0 {
		log.Error("Sent processed block to no subscribers")
	}

	// Remove pending deposits from the deposit queue.
	for _, dep := range block.Body.Deposits {
		s.depositCache.RemovePendingDeposit(ctx, dep)
	}
	return nil
}

// This checks if the block is from a competing chain, emits warning and updates metrics.
func isCompetingBlock(root []byte, slot uint64, headRoot []byte, headSlot uint64) {
	if !bytes.Equal(root[:], headRoot) {
		log.WithFields(logrus.Fields{
			"blkSlot":  slot,
			"blkRoot":  hex.EncodeToString(root[:]),
			"headSlot": headSlot,
			"headRoot": hex.EncodeToString(headRoot),
		}).Warn("Calculated head diffs from new block")
		competingBlks.Inc()
	}
}
