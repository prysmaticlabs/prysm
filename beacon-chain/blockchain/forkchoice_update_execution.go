package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/features"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"github.com/sirupsen/logrus"
)

func (s *Service) isNewProposer(slot primitives.Slot) bool {
	_, _, ok := s.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(slot, [32]byte{} /* root */)
	return ok || features.Get().PrepareAllPayloads
}

func (s *Service) isNewHead(r [32]byte) bool {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	currentHeadRoot := s.originBlockRoot
	if s.head != nil {
		currentHeadRoot = s.headRoot()
	}

	return r != currentHeadRoot || r == [32]byte{}
}

func (s *Service) getStateAndBlock(ctx context.Context, r [32]byte) (state.BeaconState, interfaces.ReadOnlySignedBeaconBlock, error) {
	if !s.hasBlockInInitSyncOrDB(ctx, r) {
		return nil, nil, errors.New("block does not exist")
	}
	newHeadBlock, err := s.getBlock(ctx, r)
	if err != nil {
		return nil, nil, err
	}
	headState, err := s.cfg.StateGen.StateByRoot(ctx, r)
	if err != nil {
		return nil, nil, err
	}
	return headState, newHeadBlock, nil
}

// fockchoiceUpdateWithExecution is a wrapper around notifyForkchoiceUpdate. It decides whether a new call to FCU should be made.
func (s *Service) forkchoiceUpdateWithExecution(ctx context.Context, newHeadRoot [32]byte, proposingSlot primitives.Slot) error {
	isNewHead := s.isNewHead(newHeadRoot)
	if !isNewHead {
		return nil
	}
	isNewProposer := s.isNewProposer(proposingSlot)
	if isNewProposer && !features.Get().DisableReorgLateBlocks {
		if s.shouldOverrideFCU(newHeadRoot, proposingSlot) {
			return nil
		}
	}
	headState, headBlock, err := s.getStateAndBlock(ctx, newHeadRoot)
	if err != nil {
		log.WithError(err).Error("Could not get forkchoice update argument")
		return nil
	}

	_, err = s.notifyForkchoiceUpdate(ctx, &notifyForkchoiceUpdateArg{
		headState: headState,
		headRoot:  newHeadRoot,
		headBlock: headBlock.Block(),
	})
	if err != nil {
		return errors.Wrap(err, "could not notify forkchoice update")
	}

	if err := s.saveHead(ctx, newHeadRoot, headBlock, headState); err != nil {
		log.WithError(err).Error("could not save head")
	}

	// Only need to prune attestations from pool if the head has changed.
	if err := s.pruneAttsFromPool(headBlock); err != nil {
		log.WithError(err).Error("could not prune attestations from pool")
	}
	return nil
}

// shouldOverrideFCU checks whether the incoming block is still subject to being
// reorged or not by the next proposer.
func (s *Service) shouldOverrideFCU(newHeadRoot [32]byte, proposingSlot primitives.Slot) bool {
	headWeight, err := s.cfg.ForkChoiceStore.Weight(newHeadRoot)
	if err != nil {
		log.WithError(err).WithField("root", fmt.Sprintf("%#x", newHeadRoot)).Warn("could not determine node weight")
	}
	currentSlot := s.CurrentSlot()
	if proposingSlot == currentSlot {
		proposerHead := s.cfg.ForkChoiceStore.GetProposerHead()
		if proposerHead != newHeadRoot {
			return true
		}
		log.WithFields(logrus.Fields{
			"root":   fmt.Sprintf("%#x", newHeadRoot),
			"weight": headWeight,
		}).Infof("Attempted late block reorg aborted due to attestations at %d seconds",
			params.BeaconConfig().SecondsPerSlot)
		lateBlockFailedAttemptSecondThreshold.Inc()
	} else {
		if s.cfg.ForkChoiceStore.ShouldOverrideFCU() {
			return true
		}
		secs, err := slots.SecondsSinceSlotStart(currentSlot,
			uint64(s.genesisTime.Unix()), uint64(time.Now().Unix()))
		if err != nil {
			log.WithError(err).Error("could not compute seconds since slot start")
		}
		if secs >= doublylinkedtree.ProcessAttestationsThreshold {
			log.WithFields(logrus.Fields{
				"root":   fmt.Sprintf("%#x", newHeadRoot),
				"weight": headWeight,
			}).Infof("Attempted late block reorg aborted due to attestations at %d seconds",
				doublylinkedtree.ProcessAttestationsThreshold)
			lateBlockFailedAttemptFirstThreshold.Inc()
		}
	}
	return false
}
