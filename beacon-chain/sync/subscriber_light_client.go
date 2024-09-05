package sync

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// TODO: event feed

func (s *Service) lightClientFinalityUpdateSubscriber(_ context.Context, msg proto.Message) error {

}

func (s *Service) lightClientOptimisticUpdateSubscriber(_ context.Context, msg proto.Message) error {

}
