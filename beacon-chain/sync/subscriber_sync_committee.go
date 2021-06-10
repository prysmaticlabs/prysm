package sync

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func (s *Service) syncCommitteeSubscriber(_ context.Context, msg proto.Message) error {
	m, ok := msg.(*eth.SyncCommitteeMessage)
	if !ok {
		return fmt.Errorf("message was not type *eth.SyncCommitteeMessage, type=%T", msg)
	}

	if m == nil {
		return errors.New("nil sync committee message")
	}

	return s.cfg.SyncCommsPool.SaveSyncCommitteeSignature(m)
}
