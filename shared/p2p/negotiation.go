package p2p

import (
	"context"

	ggio "github.com/gogo/protobuf/io"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

const handshakeProtocol = prysmProtocolPrefix + "/handshake"

// setupPeerNegotiation adds a "Connected" event handler which checks a peer's
// handshake to ensure the peer is on the same blockchain. This currently
// checks only the deposit contract address. Some peer IDs may be excluded.
// For example, a relay or bootnode will not support the handshake protocol,
// but we would not want to disconnect from those well known peer IDs.
func setupPeerNegotiation(h host.Host, contractAddress string, exclusions []peer.ID) {
	h.Network().Notify(&inet.NotifyBundle{
		ConnectedF: func(net inet.Network, conn inet.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				// Exclude bootstrap node, relay node, etc.
				for _, exclusion := range exclusions {
					if conn.RemotePeer() == exclusion {
						return
					}
				}

				log.WithField("peer", conn.RemotePeer()).Debug(
					"Checking connection to peer",
				)

				s, err := h.NewStream(
					context.Background(),
					conn.RemotePeer(),
					handshakeProtocol,
				)
				if err != nil {
					log.WithError(err).WithFields(logrus.Fields{
						"peer":    conn.RemotePeer(),
						"address": conn.RemoteMultiaddr(),
					}).Warn("Failed to open stream with newly connected peer")

					log.Warn("Temporarily disabled -- not disconnecting peer. See https://github.com/prysmaticlabs/prysm/issues/2408")
					//	if err := h.Network().ClosePeer(conn.RemotePeer()); err != nil {
					//		log.WithError(err).Error("failed to disconnect peer")
					//	}
					return
				}
				defer s.Close()

				w := ggio.NewDelimitedWriter(s)
				defer w.Close()

				hs := &pb.Handshake{DepositContractAddress: contractAddress}
				if err := w.WriteMsg(hs); err != nil {
					log.WithError(err).Error("Failed to write handshake to peer")

					if err := h.Network().ClosePeer(conn.RemotePeer()); err != nil {
						log.WithError(err).Error("failed to disconnect peer")
					}
					return
				}

				r := ggio.NewDelimitedReader(s, maxMessageSize)
				resp := &pb.Handshake{}
				if err := r.ReadMsg(resp); err != nil {
					log.WithError(err).Error("Failed to read message")

					if err := h.Network().ClosePeer(conn.RemotePeer()); err != nil {
						log.WithError(err).Error("failed to disconnect peer")
					}
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
