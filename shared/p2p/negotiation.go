package p2p

import (
	"context"

	ggio "github.com/gogo/protobuf/io"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

const handshakeProtocol = prysmProtocolPrefix + "/handshake"

// setupPeerNegotiation adds a "Connected" event handler which checks a peer's
// handshake to ensure the peer is on the same blockchain. This currently
// checks only the deposit contract address.
func setupPeerNegotiation(h host.Host, contractAddress string) {
	h.Network().Notify(&inet.NotifyBundle{
		ConnectedF: func(net inet.Network, conn inet.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				log.WithField("peer", conn.RemotePeer()).Debug(
					"Checking connection to peer",
				)

				s, err := h.NewStream(
					context.Background(),
					conn.RemotePeer(),
					handshakeProtocol,
				)
				if err != nil {
					log.WithError(err).Error(
						"Failed to open stream with newly connected peer",
					)
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

				r := ggio.NewDelimitedReader(s, maxMessageSize)
				resp := &pb.Handshake{}
				if err := r.ReadMsg(resp); err != nil {
					log.WithError(err).Error("Failed to read message")
					return
				}

				log.WithField("msg", resp).Debug("Handshake received")

				if resp.DepositContractAddress != contractAddress {
					log.WithFields(logrus.Fields{
						"peerContract":     resp.DepositContractAddress,
						"expectedContract": contractAddress,
					}).Warn("Disconnecting from peer on different contract")

					if err := h.Network().ClosePeer(conn.RemotePeer()); err != nil {
						log.WithError(err).Error("failed to disconnect peer")
					}
				}
			}()
		},
	})
}
