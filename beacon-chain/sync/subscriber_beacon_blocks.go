package sync

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

func (s *Service) beaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	signed, err := blocks.NewSignedBeaconBlock(msg)
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

	if block.Version() >= version.Deneb {
		if err := s.blockAndBlobs.addBlock(signed); err != nil {
			return err
		}
		return s.importBlockAndBlobs(ctx, root)
	}

	return s.receiveBlock(ctx, signed, root)
}

func (s *Service) importBlockAndBlobs(ctx context.Context, root [32]byte) error {
	canImport, err := s.blockAndBlobs.canImport(root)
	if err != nil {
		return err
	}
	if !canImport {
		return nil
	}
	return s.receiveBlockAndBlobs(ctx, root)
}

func (s *Service) receiveBlockAndBlobs(ctx context.Context, root [32]byte) error {
	signed, err := s.blockAndBlobs.getBlock(root)
	if err != nil {
		return err
	}

	if err = s.receiveBlock(ctx, signed, root); err != nil {
		return err
	}
	kzgs, err := signed.Block().Body().BlobKzgCommitments()
	if err != nil {
		return err
	}

	scs := make([]*eth.BlobSidecar, len(kzgs))
	for i := 0; i < len(kzgs); i++ {
		index := uint64(i)
		scs[i], err = s.blockAndBlobs.getBlob(root, index)
		if err != nil {
			return err
		}
	}

	if len(scs) > 0 {
		log.WithFields(logrus.Fields{
			"slot":      scs[0].Slot,
			"root":      fmt.Sprintf("%#x", scs[0].BlockRoot),
			"blobCount": len(scs),
		}).Info("Saving blobs")
		if err := s.cfg.beaconDB.SaveBlobSidecar(ctx, &eth.BlobSidecars{
			Sidecars: scs,
		}); err != nil {
			return err
		}
	}

	s.blockAndBlobs.delete(root)

	return nil
}

func (s *Service) receiveBlock(ctx context.Context, signed interfaces.ReadOnlySignedBeaconBlock, root [32]byte) error {
	if err := s.cfg.chain.ReceiveBlock(ctx, signed, root); err != nil {
		if blockchain.IsInvalidBlock(err) {
			r := blockchain.InvalidBlockRoot(err)
			if r != [32]byte{} {
				s.setBadBlock(ctx, r) // Setting head block as bad.
			} else {
				s.setBadBlock(ctx, root)
			}
		}
		// Set the returned invalid ancestors as bad.
		for _, root := range blockchain.InvalidAncestorRoots(err) {
			s.setBadBlock(ctx, root)
		}
		return err
	}
	return nil
}
