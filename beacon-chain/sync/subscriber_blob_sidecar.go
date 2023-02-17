package sync

import (
	"context"
	"fmt"

	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func (s *Service) blobSubscriber(ctx context.Context, msg proto.Message) error {
	b, ok := msg.(*eth.SignedBlobSidecar)
	if !ok {
		return fmt.Errorf("message was not type *eth.Attestation, type=%T", msg)
	}

	s.blobCache.insertBlob(b)

	return nil
}
