package sync

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	opfeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/operation"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func (s *Service) blobSubscriber(ctx context.Context, msg proto.Message) error {
	b, ok := msg.(*eth.SignedBlobSidecar)
	if !ok {
		return fmt.Errorf("message was not type *eth.SignedBlobSidecar, type=%T", msg)
	}

	s.setSeenBlobIndex(b.Message.Blob, b.Message.Index)

	if err := s.cfg.beaconDB.SaveBlobSidecar(ctx, []*eth.BlobSidecar{b.Message}); err != nil {
		return err
	}

	s.cfg.chain.SendNewBlobEvent([32]byte(b.Message.BlockRoot), b.Message.Index)

	s.cfg.operationNotifier.OperationFeed().Send(&feed.Event{
		Type: opfeed.BlobSidecarReceived,
		Data: &opfeed.BlobSidecarReceivedData{
			Blob: b,
		},
	})
	return nil
}
