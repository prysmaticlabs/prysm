package sync

import (
	"context"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// metaDataHandler reads the incoming metadata rpc request from the peer.
func (s *Service) metaDataHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer func() {
		if err := stream.Close(); err != nil {
			log.WithError(err).Error("Failed to close stream")
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, ttfbTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)

	if err := s.rateLimiter.validateRequest(stream, 1); err != nil {
		return err
	}
	s.rateLimiter.add(stream, 1)

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		return err
	}
	_, err := s.p2p.Encoding().EncodeWithMaxLength(stream, s.p2p.Metadata())
	return err
}

func (s *Service) sendMetaDataRequest(ctx context.Context, id peer.ID) (*pb.MetaData, error) {
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()

	stream, err := s.p2p.Send(ctx, new(interface{}), p2p.RPCMetaDataTopic, id)
	if err != nil {
		return nil, err
	}
	SetStreamReadDeadline(stream, helpers.EOFTimeout)
	code, errMsg, err := ReadStatusCode(stream, s.p2p.Encoding())
	if err != nil {
		return nil, err
	}
	if code != 0 {
		s.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		return nil, errors.New(errMsg)
	}
	msg := new(pb.MetaData)
	if err := s.p2p.Encoding().DecodeWithMaxLength(stream, msg); err != nil {
		return nil, err
	}
	return msg, nil
}
