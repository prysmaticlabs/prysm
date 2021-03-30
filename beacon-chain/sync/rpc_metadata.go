package sync

import (
	"context"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// metaDataHandler reads the incoming metadata rpc request from the peer.
func (s *Service) metaDataHandler(_ context.Context, _ interface{}, stream libp2pcore.Stream) error {
	SetRPCStreamDeadlines(stream)

	if err := s.rateLimiter.validateRequest(stream, 1); err != nil {
		return err
	}
	s.rateLimiter.add(stream, 1)

	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		return err
	}
	_, err := s.cfg.P2P.Encoding().EncodeWithMaxLength(stream, s.cfg.P2P.Metadata())
	if err != nil {
		return err
	}
	closeStream(stream, log)
	return nil
}

func (s *Service) sendMetaDataRequest(ctx context.Context, id peer.ID) (*pb.MetaData, error) {
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()

	stream, err := s.cfg.P2P.Send(ctx, new(interface{}), p2p.RPCMetaDataTopic, id)
	if err != nil {
		return nil, err
	}
	defer closeStream(stream, log)
	code, errMsg, err := ReadStatusCode(stream, s.cfg.P2P.Encoding())
	if err != nil {
		return nil, err
	}
	if code != 0 {
		s.cfg.P2P.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		return nil, errors.New(errMsg)
	}
	msg := new(pb.MetaData)
	if err := s.cfg.P2P.Encoding().DecodeWithMaxLength(stream, msg); err != nil {
		return nil, err
	}
	return msg, nil
}
