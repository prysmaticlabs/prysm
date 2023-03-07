package blockchain

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

func (s *Service) isNewProposer(slot primitives.Slot) bool {
	_, _, ok := s.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(slot, [32]byte{} /* root */)
	return ok
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

func (s *Service) forkchoiceUpdateWithExecution(ctx context.Context, newHeadRoot [32]byte, proposingSlot primitives.Slot) error {
	isNewHead := s.isNewHead(newHeadRoot)
	isNewProposer := s.isNewProposer(proposingSlot)
	if !isNewHead && !isNewProposer {
		return nil
	}

	var headState state.BeaconState
	var headBlock interfaces.ReadOnlySignedBeaconBlock
	var headRoot [32]byte
	var err error

	shouldUpdate := isNewHead
	if isNewHead && isNewProposer && !features.Get().DisableReorgLateBlocks {
		if proposingSlot == s.CurrentSlot() {
			proposerHead := s.ForkChoicer().GetProposerHead()
			if proposerHead != newHeadRoot {
				shouldUpdate = false
			}
		} else if s.ForkChoicer().ShouldOverrideFCU() {
			shouldUpdate = false
		}
	}
	if shouldUpdate {
		headRoot = newHeadRoot
		headState, headBlock, err = s.getStateAndBlock(ctx, newHeadRoot)
		if err != nil {
			log.WithError(err).Error("Could not get forkchoice update argument")
			return nil
		}
	} else {
		// We are guaranteed that the head block is the parent
		// of the incoming block. We do not process the slot
		// because it will be processed anyway in notifyForkchoiceUpdate
		headState = s.headState(ctx)
		headRoot = s.headRoot()
		headBlock, err = s.headBlock()
		if err != nil {
			return errors.Wrap(err, "could not get head block")
		}
	}

	_, err = s.notifyForkchoiceUpdate(ctx, &notifyForkchoiceUpdateArg{
		headState: headState,
		headRoot:  headRoot,
		headBlock: headBlock.Block(),
	})
	if err != nil {
		return errors.Wrap(err, "could not notify forkchoice update")
	}

	if shouldUpdate {
		if err := s.saveHead(ctx, newHeadRoot, headBlock, headState); err != nil {
			log.WithError(err).Error("could not save head")
		}

		// Only need to prune attestations from pool if the head has changed.
		if err := s.pruneAttsFromPool(headBlock); err != nil {
			return err
		}
	}

	return nil
}
