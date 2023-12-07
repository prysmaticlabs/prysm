package sync

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"google.golang.org/protobuf/proto"
)

func (s *Service) blobSubscriber(ctx context.Context, msg proto.Message) error {
	b, ok := msg.(blocks.VerifiedROBlob)
	if !ok {
		return fmt.Errorf("message was not type blocks.ROBlob, type=%T", msg)
	}

	s.setSeenBlobIndex(b.Slot(), b.ProposerIndex(), b.Index)

	if err := s.cfg.chain.ReceiveBlob(ctx, b); err != nil {
		return err
	}

	// TODO: convert operations feed to use ROBlob.
	/*
		s.cfg.operationNotifier.OperationFeed().Send(&feed.Event{
			Type: opfeed.BlobSidecarReceived,
			Data: &opfeed.BlobSidecarReceivedData{
				Blob: b,
			},
		})
	*/
	return nil
}
