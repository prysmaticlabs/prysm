package p2p

import (
	"context"

	ggio "github.com/gogo/protobuf/io"
	iconnmgr "github.com/libp2p/go-libp2p-interface-connmgr"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

const handshakeProtocol = prysmProtocolPrefix + "/handshake"

var _ = iconnmgr.ConnManager(&negotiator{})

type negotiator struct {
	contractAddress string
}

// Notifee ...
func (n *negotiator) Notifee() inet.Notifiee {
	return &inet.NotifyBundle{
		ConnectedF: func(net inet.Network, conn inet.Conn) {
			go func() {
				log.Debug("Checking connection to peer")

				s, err := net.NewStream(context.Background(), conn.RemotePeer())
				if err != nil {
					log.WithError(err).Error("Failed to open stream with newly connected peer")
					return
				}
				defer s.Close()
				s.SetProtocol(handshakeProtocol)

				w := ggio.NewDelimitedWriter(s)
				defer w.Close()

				hs := &pb.Handshake{DepositContractAddress: n.contractAddress}
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

				log.Printf("Handshake received: %v", resp)
			}()
		},
	}
}

// Unimplemented / unused interface methods.

// TagPeer is unimplemented.
func (_ negotiator) TagPeer(peer.ID, string, int) {}

// UntagPeer is unimplemented.
func (_ negotiator) UntagPeer(peer.ID, string) {}

// GetTagInfo is unimplemented.
func (_ negotiator) GetTagInfo(peer.ID) *iconnmgr.TagInfo { return &iconnmgr.TagInfo{} }

// TrimOpenConns is unimplemented.
func (_ negotiator) TrimOpenConns(context.Context) {}

// Protect is unimplemented.
func (_ negotiator) Protect(peer.ID, string) {}

// Unprotect  is unimplemented.
func (_ negotiator) Unprotect(peer.ID, string) bool { return false }
