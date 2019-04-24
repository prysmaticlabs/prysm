package p2p

import (
	"context"

	ggio "github.com/gogo/protobuf/io"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

const handshakeProtocol = prysmProtocolPrefix + "/handshake"

func setupPeerNegotiation(h host.Host, contractAddress string) {
	h.Network().Notify(&inet.NotifyBundle{
		ConnectedF: func(net inet.Network, conn inet.Conn) {
			log.Debug("Checking connection to peer")

			s, err := h.NewStream(context.Background(), conn.RemotePeer(), handshakeProtocol)
			if err != nil {
				log.WithError(err).Error("Failed to open stream with newly connected peer")
				return
			}
			defer s.Close()

			w := ggio.NewDelimitedWriter(s)
			defer w.Close()

			hs := &pb.Handshake{DepositContractAddress: contractAddress}
			if err := w.WriteMsg(hs); err != nil {
				log.WithError(err).Error("Failed to write handshake to peer")
				return
			}

			log.Info("wrote to peer")

			// TODO This read is currently blocking, it seems that the peer never
			// receives the on their handler for this protocol ID. Stepping through
			// with debugger confirms that the stream handler is never called.
			r := ggio.NewDelimitedReader(s, maxMessageSize)
			resp := &pb.Handshake{}
			if err := r.ReadMsg(resp); err != nil {
				log.WithError(err).Error("Failed to read message")
				return
			}

			log.Infof("Handshake received: %v", resp)
		},
	})
}
