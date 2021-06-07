package eventsv1

import (
	"time"

	gwpb "github.com/grpc-ecosystem/grpc-gateway/v2/proto/gateway"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/anypb"
)

func (s *Server) StreamEvents(
	req *ethpb.StreamEventsRequest, stream ethpb.Events_StreamEventsServer,
) error {
	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Info("Request: ", req)
			data, err := anypb.New(req)
			if err != nil {
				log.WithError(err).Error("Could not parse request from pb")
				return err
			}
			if err := stream.Send(&gwpb.EventSource{
				Event: "pong",
				Data:  data,
			}); err != nil {
				return err
			}
		case <-s.Ctx.Done():
			return errors.New("context canceled")
		case <-stream.Context().Done():
			return errors.New("context canceled")
		}
	}
}
