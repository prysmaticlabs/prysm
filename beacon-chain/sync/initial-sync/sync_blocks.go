package initialsync

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/libp2p/go-libp2p-peer"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// processBlock is the main method that validates each block which is received
// for initial sync. It checks if the blocks are valid and then will continue to
// process and save it into the db.
func (s *InitialSync) processBlock(ctx context.Context, block *pb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.sync.initial-sync.processBlock")
	defer span.End()
	recBlock.Inc()

	if block.Slot == s.highestObservedSlot {
		s.currentSlot = s.highestObservedSlot
		if err := s.exitInitialSync(s.ctx, block); err != nil {
			log.Errorf("Could not exit initial sync: %v", err)
			return err
		}
		return nil
	}

	if block.Slot < s.currentSlot {
		return nil
	}

	if err := s.validateAndSaveNextBlock(ctx, block); err != nil {
		return err
	}

	return nil
}

// processBatchedBlocks processes all the received blocks from
// the p2p message.
func (s *InitialSync) processBatchedBlocks(msg p2p.Message) error {
	ctx, span := trace.StartSpan(msg.Ctx, "beacon-chain.sync.initial-sync.processBatchedBlocks")
	defer span.End()
	batchedBlockReq.Inc()

	response := msg.Data.(*pb.BatchedBeaconBlockResponse)
	batchedBlocks := response.BatchedBlocks
	if len(batchedBlocks) == 0 {
		// Do not process empty responses.
		s.p2p.Reputation(msg.Peer, p2p.RepPenalityInitialSyncFailure)
		return nil
	}

	log.WithField("blocks", len(batchedBlocks)).Info("Processing batched block response")
	// Sort batchBlocks in ascending order.
	sort.Slice(batchedBlocks, func(i, j int) bool {
		return batchedBlocks[i].Slot < batchedBlocks[j].Slot
	})
	for _, block := range batchedBlocks {
		if err := s.processBlock(ctx, block); err != nil {
			return err
		}
	}
	log.Debug("Finished processing batched blocks")
	return nil
}

// requestBatchedBlocks sends out a request for multiple blocks that's between finalized roots
// and head roots.
func (s *InitialSync) requestBatchedBlocks(ctx context.Context, finalizedRoot []byte, canonicalRoot []byte, peer peer.ID) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.sync.initial-sync.requestBatchedBlocks")
	defer span.End()
	sentBatchedBlockReq.Inc()

	log.WithFields(logrus.Fields{
		"finalizedBlkRoot": fmt.Sprintf("%#x", bytesutil.Trunc(finalizedRoot[:])),
		"headBlkRoot":      fmt.Sprintf("%#x", bytesutil.Trunc(canonicalRoot[:]))},
	).Debug("Requesting batched blocks")
	if err := s.p2p.Send(ctx, &pb.BatchedBeaconBlockRequest{
		FinalizedRoot: finalizedRoot,
		CanonicalRoot: canonicalRoot,
	}, peer); err != nil {
		log.Errorf("Could not send batch block request to peer %s: %v", peer.Pretty(), err)
	}
}

// validateAndSaveNextBlock will validate whether blocks received from the blockfetcher
// routine can be added to the chain.
func (s *InitialSync) validateAndSaveNextBlock(ctx context.Context, block *pb.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.sync.initial-sync.validateAndSaveNextBlock")
	defer span.End()
	if block == nil {
		return errors.New("received nil block")
	}
	root, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		return err
	}
	if err := s.checkBlockValidity(ctx, block); err != nil {
		return err
	}
	log.WithFields(logrus.Fields{
		"root": fmt.Sprintf("%#x", bytesutil.Trunc(root[:])),
		"slot": block.Slot - params.BeaconConfig().GenesisSlot,
	}).Info("Saving block")
	s.currentSlot = block.Slot

	s.mutex.Lock()
	defer s.mutex.Unlock()
	state, err := s.db.HeadState(ctx)
	if err != nil {
		return err
	}
	if err := s.chainService.VerifyBlockValidity(ctx, block, state); err != nil {
		return err
	}
	if err := s.db.SaveBlock(block); err != nil {
		return err
	}
	if err := s.db.SaveAttestationTarget(ctx, &pb.AttestationTarget{
		Slot:       block.Slot,
		BlockRoot:  root[:],
		ParentRoot: block.ParentRootHash32,
	}); err != nil {
		return fmt.Errorf("could not to save attestation target: %v", err)
	}
	state, err = s.chainService.ApplyBlockStateTransition(ctx, block, state)
	if err != nil {
		return fmt.Errorf("could not apply block state transition: %v", err)
	}
	if err := s.chainService.CleanupBlockOperations(ctx, block); err != nil {
		return err
	}
	return s.db.UpdateChainHead(ctx, block, state)
}
