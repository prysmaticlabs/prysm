package sync

import (
"fmt"
"github.com/prysmaticlabs/prysm/shared/bytesutil"
"github.com/prysmaticlabs/prysm/shared/hashutil"
"github.com/prysmaticlabs/prysm/shared/p2p"
pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
"github.com/prysmaticlabs/prysm/shared/params"
"go.opencensus.io/trace"
)

// receiveBlockAnnounce accepts a block hash, determines if we do not contain
// the block in our local DB, and then request the full block data.
func (rs *RegularSync) receiveBlockAnnounce(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveBlockAnnounce")
	defer span.End()
	recBlockAnnounce.Inc()

	data := msg.Data.(*pb.BeaconBlockAnnounce)
	h := bytesutil.ToBytes32(data.Hash[:32])

	hasBlock := rs.db.HasBlock(h)
	span.AddAttributes(trace.BoolAttribute("hasBlock", hasBlock))

	if hasBlock {
		log.Debugf("Received a root for a block that has already been processed: %#x", h)
		return nil
	}

	log.WithField("blockRoot", fmt.Sprintf("%#x", h)).Debug("Received incoming block root, requesting full block data from sender")
	// Request the full block data from peer that sent the block hash.
	if err := rs.p2p.Send(ctx, &pb.BeaconBlockRequest{Hash: h[:]}, msg.Peer); err != nil {
		log.Error(err)
		return err
	}
	sentBlockReq.Inc()
	return nil
}

// receiveBlock processes a block message from the p2p layer and requests for its
// parents recursively if they are not yet contained in the local node's persistent storage.
func (rs *RegularSync) receiveBlock(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.receiveBlock")
	defer span.End()
	recBlock.Inc()

	response := msg.Data.(*pb.BeaconBlockResponse)
	block := response.Block
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		log.Errorf("Could not hash received block: %v", err)
		return err
	}

	log.Debugf("Processing response to block request: %#x", blockRoot)
	hasBlock := rs.db.HasBlock(blockRoot)
	span.AddAttributes(trace.BoolAttribute("hasBlock", hasBlock))
	if hasBlock {
		log.Debug("Received a block that already exists. Exiting...")
		return nil
	}

	beaconState, err := rs.db.State(ctx)
	if err != nil {
		log.Errorf("Failed to get beacon state: %v", err)
		return err
	}

	span.AddAttributes(
		trace.Int64Attribute("block.Slot", int64(block.Slot)),
		trace.Int64Attribute("finalized slot", int64(beaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch)),
	)
	if block.Slot < beaconState.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch {
		log.Debug("Discarding received block with a slot number smaller than the last finalized slot")
		return nil
	}

	// We check if we have the block's parents saved locally.
	parentRoot := bytesutil.ToBytes32(block.ParentRootHash32)
	hasParent := rs.db.HasBlock(parentRoot)
	span.AddAttributes(trace.BoolAttribute("hasParent", hasParent))

	if !hasParent {
		// If we do not have the parent, we insert it into a pending block's map.
		if err := rs.insertPendingBlock(block); err != nil {
			return err
		}
		return nil
	}

	// We then process the block by passing it through the ChainService and running
	// a fork choice rule.
	beaconState, err = rs.chainService.ReceiveBlock(ctx, block)
	if err != nil {
		log.Errorf("Could not process beacon block: %v", err)
		return err
	}
	if err := rs.chainService.ApplyForkChoiceRule(ctx, block, beaconState); err != nil {
		log.Errorf("could not apply fork choice rule: %v", err)
		return err
	}

	// If the block has a child, we then clear it from the blocks pending processing
	// and call receiveBlock recursively. The recursive function call will stop once
	// the block we process no longer has children.
	if rs.hasChild(blockRoot) {
		return rs.receiveBlock(p2p.Message{})
	}




	span.AddAttributes(trace.Int64Attribute("highestObservedSlot", int64(rs.highestObservedSlot)))
	if block.Slot < rs.highestObservedSlot {
		// If we receive a block from the past AND it corresponds to
		// a parent block of a block stored in the processing cache, that means we are
		// receiving a parent block which was missing from our db.
		if childBlock, ok := rs.blocksAwaitingProcessing[blockRoot]; ok {
			log.WithField("blockRoot", fmt.Sprintf("%#x", blockRoot)).Debug("Received missing block parent")
			delete(rs.blocksAwaitingProcessing, blockRoot)
			blocksAwaitingProcessingGauge.Dec()
			beaconState, err = rs.chainService.ReceiveBlock(ctx, block)
			if err != nil {
				log.Errorf("could not process beacon block: %v", err)
				return err
			}
			if err := rs.chainService.ApplyForkChoiceRule(ctx, block, beaconState); err != nil {
				log.Errorf("could not apply fork choice rule: %v", err)
				return err
			}
			beaconState, err = rs.chainService.ReceiveBlock(ctx, childBlock)
			if err != nil {
				log.Errorf("could not process beacon block: %v", err)
				return err
			}
			if err := rs.chainService.ApplyForkChoiceRule(ctx, childBlock, beaconState); err != nil {
				log.Errorf("could not apply fork choice rule: %v", err)
				return err
			}
			log.Debug("Sent missing block parent and child to chain service for processing")
			return nil
		}
	}

	log.WithField("blockRoot", fmt.Sprintf("%#x", blockRoot)).Debug("Sending newly received block to chain service")
	beaconState, err = rs.chainService.ReceiveBlock(ctx, block)
	if err != nil {
		log.Errorf("Could not process beacon block: %v", err)
		return err
	}
	if err := rs.chainService.ApplyForkChoiceRule(ctx, block, beaconState); err != nil {
		log.Errorf("could not apply fork choice rule: %v", err)
		return err
	}
	sentBlocks.Inc()
	// We update the last observed slot to the received canonical block's slot.
	if block.Slot > rs.highestObservedSlot {
		rs.highestObservedSlot = block.Slot
	}
	return nil
}

func (rs *RegularSync) insertPendingBlock(block *pb.BeaconBlock) error {
	rs.blocksAwaitingProcessing[parentRoot] = block
	blocksAwaitingProcessingGauge.Inc()
	rs.p2p.Broadcast(ctx, &pb.BeaconBlockRequest{Hash: parentRoot[:]})
	// We update the last observed slot to the received canonical block's slot.
	if block.Slot > rs.highestObservedSlot {
		rs.highestObservedSlot = block.Slot
	}

}

func (rs *RegularSync) hasChild(blockRoot [32]byte) bool {
	if _, ok := rs.blocksAwaitingProcessing[blockRoot]; !ok {
		return false
	}
	return true
}
