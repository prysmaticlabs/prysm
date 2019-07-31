package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// BlockReceiver interface defines the methods in the blockchain service which
// directly receives a new block from other services and applies the full processing pipeline.
type BlockReceiver interface {
	CanonicalBlockFeed() *event.Feed
	ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error
	IsCanonical(slot uint64, hash []byte) bool
	UpdateCanonicalRoots(block *ethpb.BeaconBlock, root [32]byte)
}

// BlockProcessor defines a common interface for methods useful for directly applying state transitions
// to beacon blocks and generating a new beacon state from the Ethereum 2.0 core primitives.
type BlockProcessor interface {
	CleanupBlockOperations(ctx context.Context, block *ethpb.BeaconBlock) error
}

// BlockFailedProcessingErr represents a block failing a state transition function.
type BlockFailedProcessingErr struct {
	err error
}

func (b *BlockFailedProcessingErr) Error() string {
	return fmt.Sprintf("block failed processing: %v", b.err)
}

// ReceiveBlock is a function that defines the operations that are preformed on
// any block that is received from p2p layer or rpc. It performs the following actions.
func (c *ChainService) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	c.receiveBlockLock.Lock()
	defer c.receiveBlockLock.Unlock()
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlock")
	defer span.End()

	// Run block state transition and broadcast the block to other peers.
	if err := c.forkChoiceStore.OnBlock(block); err != nil {
		return fmt.Errorf("failed to process block from fork choice service: %v", err)
	}
	if err := c.SaveAndBroadcastBlock(ctx, block); err != nil {
		return fmt.Errorf(
			"could not save and broadcast beacon block with slot %d: %v",
			block.Slot, err,
		)
	}
	root, err := ssz.SigningRoot(block)
	if err != nil {
		return fmt.Errorf("failed to compute state from block head: %v", err)
	}
	log.WithFields(logrus.Fields{
		"slots": block.Slot,
		"root":  hex.EncodeToString(root[:]),
	}).Info("successful ran state transition")

	// Run fork choice for head block and head state.
	headRoot, err := c.forkChoiceStore.Head()
	if err != nil {
		return fmt.Errorf("failed to get head from fork choice service: %v", err)
	}
	headBlk, err := c.beaconDB.Block(bytesutil.ToBytes32(headRoot))
	if err != nil {
		return fmt.Errorf("failed to compute state from block head: %v", err)
	}
	headState, err := c.beaconDB.ForkChoiceState(ctx, headRoot)
	if err != nil {
		return fmt.Errorf("failed to compute state from block head: %v", err)
	}
	if err := c.beaconDB.UpdateChainHead(ctx, headBlk, headState); err != nil {
		return fmt.Errorf("failed to update head: %v", err)
	}
	log.WithFields(logrus.Fields{
		"slots": headBlk.Slot,
		"root":  hex.EncodeToString(headRoot),
	}).Info("successful ran fork choice")

	// We process the block's contained deposits, attestations, and other operations
	// and that may need to be stored or deleted from the beacon node's persistent storage.
	if err := c.CleanupBlockOperations(ctx, block); err != nil {
		return fmt.Errorf("could not process block deposits, attestations, and other operations: %v", err)
	}

	return nil
}

// SaveAndBroadcastBlock stores the block in persistent storage and then broadcasts it to
// peers via p2p. Blocks which have already been saved are not processed again via p2p, which is why
// the order of operations is important in this function to prevent infinite p2p loops.
func (c *ChainService) SaveAndBroadcastBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return fmt.Errorf("could not tree hash incoming block: %v", err)
	}
	if err := c.beaconDB.SaveBlock(block); err != nil {
		return fmt.Errorf("failed to save block: %v", err)
	}
	if err := c.beaconDB.SaveAttestationTarget(ctx, &pb.AttestationTarget{
		Slot:            block.Slot,
		BeaconBlockRoot: blockRoot[:],
		ParentRoot:      block.ParentRoot,
	}); err != nil {
		return fmt.Errorf("failed to save attestation target: %v", err)
	}
	// Announce the new block to the network.
	c.p2p.Broadcast(ctx, &pb.BeaconBlockAnnounce{
		Hash:       blockRoot[:],
		SlotNumber: block.Slot,
	})
	return nil
}

// CleanupBlockOperations processes and cleans up any block operations relevant to the beacon node
// such as attestations, exits, and deposits. We update the latest seen attestation by validator
// in the local node's runtime, cleanup and remove pending deposits which have been included in the block
// from our node's local cache, and process validator exits and more.
func (c *ChainService) CleanupBlockOperations(ctx context.Context, block *ethpb.BeaconBlock) error {
	// Forward processed block to operation pool to remove individual operation from DB.
	if c.opsPoolService.IncomingProcessedBlockFeed().Send(block) == 0 {
		log.Error("Sent processed block to no subscribers")
	}

	if err := c.attsService.BatchUpdateLatestAttestation(ctx, block.Body.Attestations); err != nil {
		return fmt.Errorf("failed to update latest attestation for store: %v", err)
	}

	// Remove pending deposits from the deposit queue.
	for _, dep := range block.Body.Deposits {
		c.beaconDB.RemovePendingDeposit(ctx, dep)
	}
	return nil
}
