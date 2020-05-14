package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

// pingHandler reads the incoming ping rpc message from the peer.
func (r *Service) pingHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	defer func() {
		if err := stream.Close(); err != nil {
			log.WithError(err).Error("Failed to close stream")
		}
	}()
	_, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	setRPCStreamDeadlines(stream)

	m, ok := msg.(*uint64)
	if !ok {
		return fmt.Errorf("wrong message type for ping, got %T, wanted *uint64", msg)
	}
	valid, err := r.validateSequenceNum(*m, stream.Conn().RemotePeer())
	if err != nil {
		return err
	}
	if !valid {
		// send metadata request in a new routine and stream.
		go func() {
			md, err := r.sendMetaDataRequest(ctx, stream.Conn().RemotePeer())
			if err != nil {
				log.WithField("peer", stream.Conn().RemotePeer()).WithError(err).Error("Failed to send metadata request")
				return
			}
			// update metadata if there is no error
			r.p2p.Peers().SetMetadata(stream.Conn().RemotePeer(), md)
		}()
	}
	if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
		return err
	}
	_, err = r.p2p.Encoding().EncodeWithLength(stream, r.p2p.MetadataSeq())
	return err
}

func (r *Service) sendPingRequest(ctx context.Context, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	metadataSeq := r.p2p.MetadataSeq()
	stream, err := r.p2p.Send(ctx, &metadataSeq, p2p.RPCPingTopic, id)
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
		return err
	}
	valid, err := r.validateSequenceNum(*msg, stream.Conn().RemotePeer())
	if err != nil {
		r.p2p.Peers().IncrementBadResponses(stream.Conn().RemotePeer())
		return err
	}
	if valid {
		return nil
	}
	md, err := r.sendMetaDataRequest(ctx, stream.Conn().RemotePeer())
	if err != nil {
		// do not increment bad responses, as its
		// already done in the request method.
		return err
	}
	r.p2p.Peers().SetMetadata(stream.Conn().RemotePeer(), md)
	return nil
}

// validates the peer's sequence number.
func (r *Service) validateSequenceNum(seq uint64, id peer.ID) (bool, error) {
	md, err := r.p2p.Peers().Metadata(id)
	if err != nil {
		return false, err
	}
	if md == nil {
		return false, nil
	}
	if md.SeqNumber != seq {
		return false, nil
	}
	return true, nil
}
