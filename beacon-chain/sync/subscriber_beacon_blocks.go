package sync

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/config/features"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"google.golang.org/protobuf/proto"
)

func (s *Service) beaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	signed, err := blockFromProto(msg)
	if err != nil {
		return err
	}
	if err := helpers.BeaconBlockIsNil(signed); err != nil {
		return err
	}

	s.setSeenBlockIndexSlot(signed.Block().Slot(), signed.Block().ProposerIndex())

	block := signed.Block()

	root, err := block.HashTreeRoot()
	if err != nil {
		return err
	}

	if err := s.cfg.chain.ReceiveBlock(ctx, signed, root); err != nil {
		interop.WriteBlockToDisk(signed, true /*failed*/)
		s.setBadBlock(ctx, root)
		return err
	}

	if !features.Get().CorrectlyPruneCanonicalAtts {
		// Delete attestations from the block in the pool to avoid inclusion in future block.
		if err := s.deleteAttsInPool(block.Body().Attestations()); err != nil {
			log.Debugf("Could not delete attestations in pool: %v", err)
			return nil
		}
	}
	return err
}

// The input attestations are seen by the network, this deletes them from pool
// so proposers don't include them in a block for the future.
func (s *Service) deleteAttsInPool(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if helpers.IsAggregated(att) {
			if err := s.cfg.attPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			// Ideally there's shouldn't be any unaggregated attestation in the block.
			if err := s.cfg.attPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
	}
	return nil
}

func blockFromProto(msg proto.Message) (block.SignedBeaconBlock, error) {
	switch t := msg.(type) {
	case *ethpb.SignedBeaconBlock:
		return wrapper.WrappedPhase0SignedBeaconBlock(t), nil
	case *ethpb.SignedBeaconBlockAltair:
		return wrapper.WrappedAltairSignedBeaconBlock(t)
	default:
		return nil, errors.Errorf("message has invalid underlying type: %T", msg)
	}
}
