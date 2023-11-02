package blockchain

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// SendNewBlobEvent sends a message to the BlobNotifier channel that the blob
// for the blocroot `root` is ready in the database
func (s *Service) sendNewBlobEvent(root [32]byte, index uint64) {
	s.blobNotifiers.notifyIndex(root, index)
}

// ReceiveBlob saves the blob to database and sends the new event
func (s *Service) ReceiveBlob(ctx context.Context, b *ethpb.DeprecatedBlobSidecar) error {
	if err := s.cfg.BeaconDB.SaveBlobSidecar(ctx, []*ethpb.DeprecatedBlobSidecar{b}); err != nil {
		return err
	}

	s.sendNewBlobEvent([32]byte(b.BlockRoot), b.Index)
	return nil
}
