package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
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
}

// AttestationReceiver interface defines the methods in the blockchain service which
// directly receives a new attestation from other services and applies the full processing pipeline.
type AttestationReceiver interface {
	ReceiveAttestation(ctx context.Context, att *ethpb.Attestation) error
}

// BlockProcessor defines a common interface for methods useful for directly applying state transitions
// to beacon blocks and generating a new beacon state from the Ethereum 2.0 core primitives.
type BlockProcessor interface {
	CleanupBlockOperations(ctx context.Context, block *ethpb.BeaconBlock) error
}

// HeadRetriever defines a common interface for methods in blockchain service which
// directly retrieves chain head related data.
type HeadRetriever interface {
	FinalizedBlock() (*ethpb.BeaconBlock, error)
	FinalizedState(ctx context.Context) (*pb.BeaconState, error)
	FinalizedCheckpt() *ethpb.Checkpoint
	JustifiedCheckpt() *ethpb.Checkpoint
	HeadSlot() uint64
	HeadRoot() []byte
	HeadBlock() (*ethpb.BeaconBlock, error)
	HeadState() (*pb.BeaconState, error)
	CanonicalRoot(slot uint64) []byte
}

// BlockFailedProcessingErr represents a block failing a state transition function.yea
type BlockFailedProcessingErr struct {
	err error
}

func (b *BlockFailedProcessingErr) Error() string {
	return fmt.Sprintf("block failed processing: %v", b.err)
}

// ReceiveBlock is a function that defines the operations that are preformed on
// any block that is received from p2p layer or rpc.
func (c *ChainService) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlock")
	defer span.End()

	root, err := ssz.SigningRoot(block)
	if err != nil {
		return errors.Wrap(err, "could not get signing root on received block")
	}

	// Update fork choice service's time tracker to current time.
	c.forkChoiceStore.OnTick(uint64(time.Now().Unix()))

	// Run block state transition and broadcast the block to other peers.
	if err := c.forkChoiceStore.OnBlock(block); err != nil {
		c.beaconDB.MarkEvilBlockHash(root)
		return errors.Wrap(err, "could not process block from fork choice service")
	}
	log.WithFields(logrus.Fields{
		"slots": block.Slot,
		"root":  hex.EncodeToString(root[:]),
	}).Info("Finished state transition and updated store for block")

	// Announce the new block to the network.
	c.p2p.Broadcast(ctx, &pb.BeaconBlockAnnounce{
		Hash:       root[:],
		SlotNumber: block.Slot,
	})

	// Run fork choice for head block and head block.
	headRoot, err := c.forkChoiceStore.Head()
	if err != nil {
		return errors.Wrap(err, "could not get head from fork choice service")
	}
	headBlk, err := c.beaconDB.Block(bytesutil.ToBytes32(headRoot))
	if err != nil {
		return errors.Wrap(err, "could not compute state from block head")
	}

	c.canonicalRootsLock.Lock()
	defer c.canonicalRootsLock.Unlock()
	c.headSlot = headBlk.Slot
	c.canonicalRoots[headBlk.Slot] = headRoot
	if err := c.beaconDB.SaveHeadBlockRoot(headRoot); err != nil {
		return errors.Wrap(err, "could not save head root in DB")
	}

	log.WithFields(logrus.Fields{
		"slots": headBlk.Slot,
		"root":  hex.EncodeToString(headRoot),
	}).Info("Finished fork choice for block")

	// We process the block's contained deposits, attestations, and other operations
	// and that may need to be stored or deleted from the beacon node's persistent storage.
	if err := c.CleanupBlockOperations(ctx, block); err != nil {
		return errors.Wrap(err, "could not clean up block deposits, attestations, and other operations")
	}

	return nil
}

// ReceiveAttestation is a function that defines the operations that are preformed on
// any attestation that is received from p2p layer or rpc.
func (c *ChainService) ReceiveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveAttestation")
	defer span.End()

	root, err := ssz.SigningRoot(att)
	if err != nil {
		return errors.Wrap(err, "could not sign root attestation")
	}

	// broadcast the attestation to other peers.
	c.p2p.Broadcast(ctx, att)

	// Delay attestation inclusion until the attested slot is in the past.
	if err := waitForAttInclDelay(ctx, c.beaconDB, att); err != nil {
		return errors.Wrap(err, "could not delay attestation inclusion")
	}

	c.forkChoiceStore.OnTick(uint64(time.Now().Unix()))

	if err := c.forkChoiceStore.OnAttestation(att); err != nil {
		return errors.Wrap(err, "could not process block from fork choice service")
	}
	log.WithFields(logrus.Fields{
		"root": hex.EncodeToString(root[:]),
	}).Info("Finished update fork choice store for attestation")

	// Run fork choice for head block and head block.
	headRoot, err := c.forkChoiceStore.Head()
	if err != nil {
		return errors.Wrap(err, "could not get head from fork choice service")
	}
	headBlk, err := c.beaconDB.Block(bytesutil.ToBytes32(headRoot))
	if err != nil {
		return errors.Wrap(err, "could not compute state from block head")

	}

	c.canonicalRootsLock.Lock()
	defer c.canonicalRootsLock.Unlock()
	c.headSlot = headBlk.Slot
	c.canonicalRoots[headBlk.Slot] = headRoot
	if err := c.beaconDB.SaveHeadBlockRoot(headRoot); err != nil {
		return errors.Wrap(err, "could not save head root in DB")
	}

	log.WithFields(logrus.Fields{
		"slots": headBlk.Slot,
		"root":  hex.EncodeToString(headRoot),
	}).Info("Finished fork choice for attestation")

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

	// Remove pending deposits from the deposit queue.
	for _, dep := range block.Body.Deposits {
		c.beaconDB.RemovePendingDeposit(ctx, dep)
	}
	return nil
}
