package sync

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"google.golang.org/protobuf/proto"
)

func (s *Service) beaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	rBlock, ok := msg.(*ethpb.SignedBeaconBlock)
	if !ok {
		return errors.New("message is not type *ethpb.SignedBeaconBlock")
	}
	signed := wrapper.WrappedPhase0SignedBeaconBlock(rBlock)

	if signed.IsNil() || signed.Block().IsNil() {
		return errors.New("nil block")
	}

	s.setSeenBlockIndexSlot(signed.Block().Slot(), signed.Block().ProposerIndex())

	block := signed.Block()

	root, err := block.HashTreeRoot()
	if err != nil {
		return err
	}

	if err := s.cfg.Chain.ReceiveBlock(ctx, signed, root); err != nil {
		interop.WriteBlockToDisk(signed, true /*failed*/)
		s.setBadBlock(ctx, root)
		return err
	}

	return nil
}
