package blockchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// BlockReceiver interface defines the methods in the blockchain service which
// directly receives a new block from other services and applies the full processing pipeline.
type BlockReceiver interface {
	CanonicalBlockFeed() *event.Feed
	ReceiveBlock(ctx context.Context, block *pb.BeaconBlock) (*pb.BeaconState, error)
	IsCanonical(slot uint64, hash []byte) bool
	CanonicalBlock(slot uint64) (*pb.BeaconBlock, error)
	RecentCanonicalRoots(count uint64) []*pbrpc.BlockRoot
}

// BlockProcessor defines a common interface for methods useful for directly applying state transitions
// to beacon blocks and generating a new beacon state from the Ethereum 2.0 core primitives.
type BlockProcessor interface {
	VerifyBlockValidity(ctx context.Context, block *pb.BeaconBlock, beaconState *pb.BeaconState) error
	ApplyBlockStateTransition(ctx context.Context, block *pb.BeaconBlock, beaconState *pb.BeaconState) (*pb.BeaconState, error)
	CleanupBlockOperations(ctx context.Context, block *pb.BeaconBlock) error
}

// BlockFailedProcessingErr represents a block failing a state transition function.
type BlockFailedProcessingErr struct {
	err error
}

func (b *BlockFailedProcessingErr) Error() string {
	return fmt.Sprintf("block failed processing: %v", b.err)
}

// ReceiveBlock is a function that defines the operations that are preformed on
// any block that is received from p2p layer or rpc. It performs the following actions: It checks the block to see
// 1. Verify a block passes pre-processing conditions
// 2. Save and broadcast the block via p2p to other peers
// 3. Apply the block state transition function and account for skip slots.
// 4. Process and cleanup any block operations, such as attestations and deposits, which would need to be
//    either included or flushed from the beacon node's runtime.
func (c *ChainService) ReceiveBlock(ctx context.Context, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	c.receiveBlockLock.Lock()
	defer c.receiveBlockLock.Unlock()
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveBlock")
	defer span.End()
	parentRoot := bytesutil.ToBytes32(block.ParentRootHash32)
	parent, err := c.beaconDB.Block(parentRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent block: %v", err)
	}
	if parent == nil {
		return nil, errors.New("parent does not exist in DB")
	}
	beaconState, err := c.beaconDB.HistoricalStateFromSlot(ctx, parent.Slot)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve beacon state: %v", err)
	}
	saveLatestBlock := beaconState.LatestBlock

	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return nil, fmt.Errorf("could not hash beacon block")
	}
	// We first verify the block's basic validity conditions.
	if err := c.VerifyBlockValidity(ctx, block, beaconState); err != nil {
		return beaconState, fmt.Errorf("block with slot %d is not ready for processing: %v", block.Slot, err)
	}

	// We save the block to the DB and broadcast it to our peers.
	if err := c.SaveAndBroadcastBlock(ctx, block); err != nil {
		return beaconState, fmt.Errorf(
			"could not save and broadcast beacon block with slot %d: %v",
			block.Slot-params.BeaconConfig().GenesisSlot, err,
		)
	}

	log.WithField("slotNumber", block.Slot-params.BeaconConfig().GenesisSlot).Info(
		"Executing state transition")

	// We then apply the block state transition accordingly to obtain the resulting beacon state.
	beaconState, err = c.ApplyBlockStateTransition(ctx, block, beaconState)
	if err != nil {
		switch err.(type) {
		case *BlockFailedProcessingErr:
			// If the block fails processing, we mark it as blacklisted and delete it from our DB.
			c.beaconDB.MarkEvilBlockHash(blockRoot)
			if err := c.beaconDB.DeleteBlock(block); err != nil {
				return nil, fmt.Errorf("could not delete bad block from db: %v", err)
			}
			return beaconState, err
		default:
			return beaconState, fmt.Errorf("could not apply block state transition: %v", err)
		}
	}

	log.WithFields(logrus.Fields{
		"slotNumber":   block.Slot - params.BeaconConfig().GenesisSlot,
		"currentEpoch": helpers.SlotToEpoch(block.Slot) - params.BeaconConfig().GenesisEpoch,
	}).Info("State transition complete")

	// Check state root
	if featureconfig.FeatureConfig().EnableCheckBlockStateRoot {
		// Calc state hash with previous block
		beaconState.LatestBlock = saveLatestBlock
		stateRoot, err := hashutil.HashProto(beaconState)
		if err != nil {
			return nil, fmt.Errorf("could not hash beacon state: %v", err)
		}
		beaconState.LatestBlock = block
		if !bytes.Equal(block.StateRootHash32, stateRoot[:]) {
			return nil, fmt.Errorf("beacon state root is not equal to block state root: %#x != %#x", stateRoot, block.StateRootHash32)
		}
	}

	// We process the block's contained deposits, attestations, and other operations
	// and that may need to be stored or deleted from the beacon node's persistent storage.
	if err := c.CleanupBlockOperations(ctx, block); err != nil {
		return beaconState, fmt.Errorf("could not process block deposits, attestations, and other operations: %v", err)
	}

	log.WithField("slot", block.Slot-params.BeaconConfig().GenesisSlot).Info("Finished processing beacon block")
	return beaconState, nil
}

// ApplyBlockStateTransition runs the Ethereum 2.0 state transition function
// to produce a new beacon state and also accounts for skip slots occurring.
//
//  def apply_block_state_transition(block):
//  	# process skipped slots
// 		while (state.slot < block.slot - 1):
//      	state = slot_state_transition(state, block=None)
//
//		# process slot with block
//		state = slot_state_transition(state, block)
//
//		# check state root
//      if block.state_root == hash(state):
//			return state, error
//		else:
//			return nil, error  # or throw or whatever
//
func (c *ChainService) ApplyBlockStateTransition(
	ctx context.Context, block *pb.BeaconBlock, beaconState *pb.BeaconState,
) (*pb.BeaconState, error) {
	// Retrieve the last processed beacon block's hash root.
	headRoot, err := c.ChainHeadRoot()
	if err != nil {
		return beaconState, fmt.Errorf("could not retrieve chain head root: %v", err)
	}

	// Check for skipped slots.
	numSkippedSlots := 0
	for beaconState.Slot < block.Slot-1 {
		beaconState, err = c.runStateTransition(ctx, headRoot, nil, beaconState)
		if err != nil {
			return beaconState, err
		}
		numSkippedSlots++
	}
	if numSkippedSlots > 0 {
		log.Warnf("Processed %d skipped slots", numSkippedSlots)
	}

	beaconState, err = c.runStateTransition(ctx, headRoot, block, beaconState)
	if err != nil {
		return beaconState, err
	}
	return beaconState, nil
}

// VerifyBlockValidity cross-checks the block against the pre-processing conditions from
// Ethereum 2.0, namely:
//   The parent block with root block.parent_root has been processed and accepted.
//   The node has processed its state up to slot, block.slot - 1.
//   The Ethereum 1.0 block pointed to by the state.processed_pow_receipt_root has been processed and accepted.
//   The node's local clock time is greater than or equal to state.genesis_time + block.slot * SECONDS_PER_SLOT.
func (c *ChainService) VerifyBlockValidity(
	ctx context.Context,
	block *pb.BeaconBlock,
	beaconState *pb.BeaconState,
) error {
	if block.Slot == params.BeaconConfig().GenesisSlot {
		return fmt.Errorf("cannot process a genesis block: received block with slot %d",
			block.Slot-params.BeaconConfig().GenesisSlot)
	}
	powBlockFetcher := c.web3Service.Client().BlockByHash
	if err := b.IsValidBlock(ctx, beaconState, block,
		c.beaconDB.HasBlock, powBlockFetcher, c.genesisTime); err != nil {
		return fmt.Errorf("block does not fulfill pre-processing conditions %v", err)
	}
	return nil
}

// SaveAndBroadcastBlock stores the block in persistent storage and then broadcasts it to
// peers via p2p. Blocks which have already been saved are not processed again via p2p, which is why
// the order of operations is important in this function to prevent infinite p2p loops.
func (c *ChainService) SaveAndBroadcastBlock(ctx context.Context, block *pb.BeaconBlock) error {
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return fmt.Errorf("could not tree hash incoming block: %v", err)
	}
	if err := c.beaconDB.SaveBlock(block); err != nil {
		return fmt.Errorf("failed to save block: %v", err)
	}
	if err := c.beaconDB.SaveAttestationTarget(ctx, &pb.AttestationTarget{
		Slot:       block.Slot,
		BlockRoot:  blockRoot[:],
		ParentRoot: block.ParentRootHash32,
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
func (c *ChainService) CleanupBlockOperations(ctx context.Context, block *pb.BeaconBlock) error {
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

// runStateTransition executes the Ethereum 2.0 core state transition for the beacon chain and
// updates important checkpoints and local persistent data during epoch transitions. It serves as a wrapper
// around the more low-level, core state transition function primitive.
func (c *ChainService) runStateTransition(
	ctx context.Context,
	headRoot [32]byte,
	block *pb.BeaconBlock,
	beaconState *pb.BeaconState,
) (*pb.BeaconState, error) {
	newState, err := state.ExecuteStateTransition(
		ctx,
		beaconState,
		block,
		headRoot,
		&state.TransitionConfig{
			VerifySignatures: false, // We disable signature verification for now.
			Logging:          true,  // We enable logging in this state transition call.
		},
	)
	if err != nil {
		return beaconState, &BlockFailedProcessingErr{err}
	}
	log.WithField(
		"slotsSinceGenesis", newState.Slot-params.BeaconConfig().GenesisSlot,
	).Info("Slot transition successfully processed")

	if block != nil {
		log.WithField(
			"slotsSinceGenesis", newState.Slot-params.BeaconConfig().GenesisSlot,
		).Info("Block transition successfully processed")

		// Save Historical States.
		if err := c.beaconDB.SaveHistoricalState(ctx, beaconState); err != nil {
			return nil, fmt.Errorf("could not save historical state: %v", err)
		}
	}

	if helpers.IsEpochEnd(newState.Slot) {
		// Save activated validators of this epoch to public key -> index DB.
		if err := c.saveValidatorIdx(newState); err != nil {
			return newState, fmt.Errorf("could not save validator index: %v", err)
		}
		// Delete exited validators of this epoch to public key -> index DB.
		if err := c.deleteValidatorIdx(newState); err != nil {
			return newState, fmt.Errorf("could not delete validator index: %v", err)
		}
		// Update FFG checkpoints in DB.
		if err := c.updateFFGCheckPts(ctx, newState); err != nil {
			return newState, fmt.Errorf("could not update FFG checkpts: %v", err)
		}
		log.WithField(
			"SlotsSinceGenesis", newState.Slot-params.BeaconConfig().GenesisSlot,
		).Info("Epoch transition successfully processed")
	}
	return newState, nil
}

// saveValidatorIdx saves the validators public key to index mapping in DB, these
// validators were activated from current epoch. After it saves, current epoch key
// is deleted from ActivatedValidators mapping.
func (c *ChainService) saveValidatorIdx(state *pb.BeaconState) error {
	activatedValidators := validators.ActivatedValFromEpoch(helpers.CurrentEpoch(state) + 1)
	for _, idx := range activatedValidators {
		pubKey := state.ValidatorRegistry[idx].Pubkey
		if err := c.beaconDB.SaveValidatorIndex(pubKey, int(idx)); err != nil {
			return fmt.Errorf("could not save validator index: %v", err)
		}
	}
	validators.DeleteActivatedVal(helpers.CurrentEpoch(state))
	return nil
}

// deleteValidatorIdx deletes the validators public key to index mapping in DB, the
// validators were exited from current epoch. After it deletes, current epoch key
// is deleted from ExitedValidators mapping.
func (c *ChainService) deleteValidatorIdx(state *pb.BeaconState) error {
	exitedValidators := validators.ExitedValFromEpoch(helpers.CurrentEpoch(state) + 1)
	for _, idx := range exitedValidators {
		pubKey := state.ValidatorRegistry[idx].Pubkey
		if err := c.beaconDB.DeleteValidatorIndex(pubKey); err != nil {
			return fmt.Errorf("could not delete validator index: %v", err)
		}
	}
	validators.DeleteExitedVal(helpers.CurrentEpoch(state))
	return nil
}
