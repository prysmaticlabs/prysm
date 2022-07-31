package sync

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func (s *Service) blobsSidecarSubscriber(ctx context.Context, msg proto.Message) error {
	m, ok := msg.(*ethpb.SignedBlobsSidecar)
	if !ok {
		return fmt.Errorf("message was not type *eth.SignedBlobsSidecar, type=%T", msg)
	}
	if m == nil {
		return errors.New("nil blobs sidecar message")
	}

	// Sidecars are handled by the block queue processing routine
	s.pendingQueueLock.Lock()
	s.insertSidecarToPendingQueue(&queuedBlobsSidecar{m.Message, m.Signature, true})
	s.pendingQueueLock.Unlock()
	return nil
}
