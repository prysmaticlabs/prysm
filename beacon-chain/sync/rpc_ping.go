package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	libp2pcore "github.com/libp2p/go-libp2p-core"
)

// pingHandler reads the incoming goodbye rpc message from the peer.
func (r *Service) pingHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer stream.Close()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)

	m, ok := msg.(*uint64)
	if !ok {
		return fmt.Errorf("wrong message type for ping, got %T, wanted *uint64", msg)
	}
	if err := r.validateSequenceNum(*m, stream.Conn().RemotePeer()); err != nil {
		return err
	}
	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		return err
	}
	_, err := r.p2p.Encoding().EncodeWithLength(stream, r.p2p.MetadataSeq())
	return err
}

func (r *Service) sendPingRequest(ctx context.Context, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stream, err := r.p2p.Send(ctx, r.p2p.MetadataSeq(), id)
	if err != nil {
		return err
	}

	code, errMsg, err := ReadStatusCode(stream, r.p2p.Encoding())
	if err != nil {
		return err
	}

	if code != 0 {
		r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		return errors.New(errMsg)
	}
	msg := new(uint64)
	if err := r.p2p.Encoding().DecodeWithLength(stream, msg); err != nil {
		r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		return err
	}
	err = r.validateSequenceNum(*msg, stream.Conn().RemotePeer())
	if err != nil {
		r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
	}
	return err
}

// validates the peer's sequence number.
func (r *Service) validateSequenceNum(seq uint64, id peer.ID) error {
	// no-op
	return nil
}
