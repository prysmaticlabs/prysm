package sync

import (
	"context"
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
	if _, err := r.p2p.Encoding().EncodeWithLength(stream, r.p2p.MetadataSeq()); err != nil {
		return err
	}
	return r.p2p.Disconnect(stream.Conn().RemotePeer())
}

// validates the peer's sequence number.
func (r *Service) validateSequenceNum(seq uint64, id peer.ID) error {
	// no-op
	return nil
}
