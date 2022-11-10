package sync

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func (s *Service) beaconBlockAndBlobsSubscriber(ctx context.Context, msg proto.Message) error {
	m, ok := msg.(*ethpb.SignedBeaconBlockAndBlobsSidecar)
	if !ok {
		return fmt.Errorf("message was not type *eth.SignedBeaconBlockAndBlobsSidecar, type=%T", msg)
	}
	if m == nil {
		return errors.New("nil blobs sidecar message")
	}

	signed, err := blocks.NewSignedBeaconBlock(m.BeaconBlock)
	if err != nil {
		return err
	}
	if err := blocks.BeaconBlockIsNil(signed); err != nil {
		return err
	}

	s.setSeenBlockIndexSlot(signed.Block().Slot(), signed.Block().ProposerIndex())

	block := signed.Block()

	root, err := block.HashTreeRoot()
	if err != nil {
		return err
	}

	if err := s.cfg.chain.ReceiveBlock(ctx, signed, root); err != nil {
		if blockchain.IsInvalidBlock(err) {
			r := blockchain.InvalidBlockRoot(err)
			if r != [32]byte{} {
				s.setBadBlock(ctx, r) // Setting head block as bad.
			} else {
				interop.WriteBlockToDisk(signed, true /*failed*/)
				s.setBadBlock(ctx, root)
			}
		}
		// Set the returned invalid ancestors as bad.
		for _, root := range blockchain.InvalidAncestorRoots(err) {
			s.setBadBlock(ctx, root)
		}
		return err
	}

	err = s.cfg.beaconDB.SaveBlobsSidecar(ctx, m.BlobsSidecar)
	if err != nil {
		return err
	}

	return nil
}
