package initialsync

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

func (s *InitialSync) processState(msg p2p.Message) {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.initial-sync.processState")
	defer span.End()
	data := msg.Data.(*pb.BeaconStateResponse)
	finalizedState := data.FinalizedState
	recState.Inc()

	if err := s.db.SaveFinalizedState(finalizedState); err != nil {
		log.Errorf("Unable to set received last finalized state in db: %v", err)
		return
	}

	if err := s.db.SaveHistoricalState(ctx, finalizedState); err != nil {
		log.Errorf("Could not save new historical state: %v", err)
		return
	}

	if err := s.db.SaveFinalizedBlock(finalizedState.LatestBlock); err != nil {
		log.Errorf("Could not save finalized block %v", err)
		return
	}

	if err := s.db.SaveBlock(finalizedState.LatestBlock); err != nil {
		log.Errorf("Could not save block %v", err)
		return
	}

	if err := s.db.SaveJustifiedState(finalizedState); err != nil {
		log.Errorf("Could not set beacon state for initial sync %v", err)
		return
	}

	if err := s.db.SaveJustifiedBlock(finalizedState.LatestBlock); err != nil {
		log.Errorf("Could not save finalized block %v", err)
		return
	}

	exists, blkNum, err := s.powchain.BlockExists(ctx, bytesutil.ToBytes32(finalizedState.LatestEth1Data.BlockHash32))
	if err != nil {
		log.Errorf("Unable to get powchain block %v", err)
	}

	if !exists {
		log.Error("Latest ETH1 block doesn't exist in the pow chain")
		return
	}

	s.db.PrunePendingDeposits(ctx, blkNum)

	if err := s.db.UpdateChainHead(ctx, finalizedState.LatestBlock, finalizedState); err != nil {
		log.Errorf("Could not update chain head: %v", err)
		return
	}

	// sets the current slot to the last finalized slot of the
	// beacon state to begin our sync from.
	s.currentSlot = finalizedState.Slot
	s.stateReceived = true
	log.Debugf(
		"Successfully saved beacon state with the last finalized slot: %d",
		finalizedState.Slot-params.BeaconConfig().GenesisSlot,
	)
	s.requestBatchedBlocks(s.currentSlot+1, s.highestObservedSlot)
	s.lastRequestedSlot = s.highestObservedSlot
}

// requestStateFromPeer requests for the canonical state, finalized state, and justified state from a peer.
func (s *InitialSync) requestStateFromPeer(ctx context.Context, lastFinalizedRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.sync.initial-sync.requestStateFromPeer")
	defer span.End()
	stateReq.Inc()
	return s.p2p.Send(ctx, &pb.BeaconStateRequest{
		FinalizedStateRootHash32S: lastFinalizedRoot[:],
	}, p2p.AnyPeer)
}
