package sync

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func (s *Service) blobSubscriber(ctx context.Context, msg proto.Message) error {
	b, ok := msg.(*eth.SignedBlobSidecar)
	if !ok {
		return fmt.Errorf("message was not type *eth.Attestation, type=%T", msg)
	}

	if err := s.blockAndBlobs.addBlob(b.Message); err != nil {
		return errors.Wrap(err, "could not add blob to queue")
	}

	if err := s.importBlockAndBlobs(ctx, bytesutil.ToBytes32(b.Message.BlockRoot)); err != nil {
		return errors.Wrap(err, "could not import block and blobs")
	}

	return nil
}
