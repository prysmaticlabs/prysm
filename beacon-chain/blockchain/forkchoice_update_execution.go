package blockchain

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

func (s *Service) isNewProposer() bool {
	_, _, ok := s.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(s.CurrentSlot()+1, [32]byte{} /* root */)
	return ok
}

func (s *Service) isNewHead(r [32]byte) bool {
	s.headLock.RLock()
	defer s.headLock.RUnlock()
	return r == [32]byte{} || r != s.headRoot()
}

func (s *Service) getHeadStateAndBlock(ctx context.Context, r [32]byte) (state.BeaconState, interfaces.SignedBeaconBlock, error) {
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

func (s *Service) forkchoiceUpdateWithExecution(ctx context.Context, newHeadRoot [32]byte) error {
	if !s.isNewHead(newHeadRoot) && !s.isNewProposer() {
		return nil
	}

	headState, headBlock, err := s.getHeadStateAndBlock(ctx, newHeadRoot)
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
		return err
	}

	if err := s.saveHead(ctx, newHeadRoot, headBlock, headState); err != nil {
		log.WithError(err).Error("could not save head")
	}
	return nil
}
