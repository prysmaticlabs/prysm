package sync

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (s *Service) validateBlob(ctx context.Context, id peer.ID, message *pubsub.Message) (pubsub.ValidationResult, error) {
	return pubsub.ValidationAccept, nil
}
