package sync

import (
	"context"
	"fmt"

	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func (s *Service) blobSubscriber(ctx context.Context, msg proto.Message) error {
	b, ok := msg.(*eth.SignedBlobSidecar)
	if !ok {
		return fmt.Errorf("message was not type *eth.SignedBlobSidecar, type=%T", msg)
	}

	log.WithFields(blobFields(b.Message)).Debug("Received blob sidecar")

	s.setSeenBlobIndex(b.Message.Blob, b.Message.Index)

	// TODO: Store blobs in cache. Will be addressed in subsequent PR.

	return nil
}
