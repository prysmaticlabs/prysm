// Package testing includes useful utilities for mocking
// a beacon node's p2p service for unit tests.
package testing

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/gogo/protobuf/proto"
	bhost "github.com/libp2p/go-libp2p-blankhost"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	"github.com/multiformats/go-multiaddr"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

// TestP2P represents a p2p implementation that can be used for testing.
type TestP2P struct {
	t               *testing.T
	BHost           host.Host
	pubsub          *pubsub.PubSub
	BroadcastCalled bool
	DelaySend       bool
	Digest          [4]byte
	peers           *peers.Status
	LocalMetadata   *pb.MetaData
}

// NewTestP2P initializes a new p2p test service.
func NewTestP2P(t *testing.T) *TestP2P {
	ctx := context.Background()
	h := bhost.NewBlankHost(swarmt.GenSwarm(t, ctx))
	ps, err := pubsub.NewFloodSub(ctx, h,
		pubsub.WithMessageSigning(false),
		pubsub.WithStrictSignatureVerification(false),
	)
	if err != nil {
		t.Fatal(err)
	}

	return &TestP2P{
		t:      t,
		BHost:  h,
		pubsub: ps,
		peers:  peers.NewStatus(5 /* maxBadResponses */),
	}
}

// Connect two test peers together.
func (p *TestP2P) Connect(b *TestP2P) {
	if err := connect(p.BHost, b.BHost); err != nil {
		p.t.Fatal(err)
	}
}

func connect(a, b host.Host) error {
	pinfo := b.Peerstore().PeerInfo(b.ID())
	return a.Connect(context.Background(), pinfo)
}

// ReceiveRPC simulates an incoming RPC.
func (p *TestP2P) ReceiveRPC(topic string, msg proto.Message) {
	h := bhost.NewBlankHost(swarmt.GenSwarm(p.t, context.Background()))
	if err := connect(h, p.BHost); err != nil {
		p.t.Fatalf("Failed to connect two peers for RPC: %v", err)
	}
	s, err := h.NewStream(context.Background(), p.BHost.ID(), protocol.ID(topic+p.Encoding().ProtocolSuffix()))
	if err != nil {
		p.t.Fatalf("Failed to open stream %v", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			p.t.Log(err)
		}
	}()

	n, err := p.Encoding().EncodeWithMaxLength(s, msg)
	if err != nil {
		p.t.Fatalf("Failed to encode message: %v", err)
	}

	p.t.Logf("Wrote %d bytes", n)
}

// ReceivePubSub simulates an incoming message over pubsub on a given topic.
func (p *TestP2P) ReceivePubSub(topic string, msg proto.Message) {
	h := bhost.NewBlankHost(swarmt.GenSwarm(p.t, context.Background()))
	ps, err := pubsub.NewFloodSub(context.Background(), h,
		pubsub.WithMessageSigning(false),
		pubsub.WithStrictSignatureVerification(false),
	)
	if err != nil {
		p.t.Fatalf("Failed to create flood sub: %v", err)
	}
	if err := connect(h, p.BHost); err != nil {
		p.t.Fatalf("Failed to connect two peers for RPC: %v", err)
	}

	// PubSub requires some delay after connecting for the (*PubSub).processLoop method to
	// pick up the newly connected peer.
	time.Sleep(time.Millisecond * 100)

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().EncodeGossip(buf, msg); err != nil {
		p.t.Fatalf("Failed to encode message: %v", err)
	}
	digest, err := p.ForkDigest()
	if err != nil {
		p.t.Fatal(err)
	}
	topic = fmt.Sprintf(topic, digest)
	topic = topic + p.Encoding().ProtocolSuffix()

	if err := ps.Publish(topic, buf.Bytes()); err != nil {
		p.t.Fatalf("Failed to publish message; %v", err)
	}
}

// Broadcast a message.
func (p *TestP2P) Broadcast(ctx context.Context, msg proto.Message) error {
	p.BroadcastCalled = true
	return nil
}

// BroadcastAttestation broadcasts an attestation.
func (p *TestP2P) BroadcastAttestation(ctx context.Context, subnet uint64, att *ethpb.Attestation) error {
	p.BroadcastCalled = true
	return nil
}

// SetStreamHandler for RPC.
func (p *TestP2P) SetStreamHandler(topic string, handler network.StreamHandler) {
	p.BHost.SetStreamHandler(protocol.ID(topic), handler)
}

// Encoding returns ssz encoding.
func (p *TestP2P) Encoding() encoder.NetworkEncoding {
	return &encoder.SszNetworkEncoder{}
}

// PubSub returns reference underlying floodsub. This test library uses floodsub
// to ensure all connected peers receive the message.
func (p *TestP2P) PubSub() *pubsub.PubSub {
	return p.pubsub
}

// Disconnect from a peer.
func (p *TestP2P) Disconnect(pid peer.ID) error {
	return p.BHost.Network().ClosePeer(pid)
}

// PeerID returns the Peer ID of the local peer.
func (p *TestP2P) PeerID() peer.ID {
	return p.BHost.ID()
}

// Host returns the libp2p host of the
// local peer.
func (p *TestP2P) Host() host.Host {
	return p.BHost
}

// ENR returns the enr of the local peer.
func (p *TestP2P) ENR() *enr.Record {
	return new(enr.Record)
}

// AddConnectionHandler handles the connection with a newly connected peer.
func (p *TestP2P) AddConnectionHandler(f func(ctx context.Context, id peer.ID) error,
	g func(context.Context, peer.ID) error) {
	p.BHost.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				p.peers.Add(new(enr.Record), conn.RemotePeer(), conn.RemoteMultiaddr(), conn.Stat().Direction)
				ctx := context.Background()

				p.peers.SetConnectionState(conn.RemotePeer(), peers.PeerConnecting)
				if err := f(ctx, conn.RemotePeer()); err != nil {
					logrus.WithError(err).Error("Could not send succesful hello rpc request")
					if err := p.Disconnect(conn.RemotePeer()); err != nil {
						logrus.WithError(err).Errorf("Unable to close peer %s", conn.RemotePeer())
					}
					p.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnected)
					return
				}
				p.peers.SetConnectionState(conn.RemotePeer(), peers.PeerConnected)
			}()
		},
	})
}

// AddDisconnectionHandler --
func (p *TestP2P) AddDisconnectionHandler(f func(ctx context.Context, id peer.ID) error) {
	p.BHost.Network().Notify(&network.NotifyBundle{
		DisconnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				p.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnecting)
				if err := f(context.Background(), conn.RemotePeer()); err != nil {
					logrus.WithError(err).Debug("Unable to invoke callback")
				}
				p.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnected)
			}()
		},
	})
}

// Send a message to a specific peer.
func (p *TestP2P) Send(ctx context.Context, msg interface{}, topic string, pid peer.ID) (network.Stream, error) {
	protocol := topic
	if protocol == "" {
		return nil, fmt.Errorf("protocol doesnt exist for proto message: %v", msg)
	}
	stream, err := p.BHost.NewStream(ctx, pid, core.ProtocolID(protocol+p.Encoding().ProtocolSuffix()))
	if err != nil {
		return nil, err
	}

	if topic != "/eth2/beacon_chain/req/metadata/1" {
		if _, err := p.Encoding().EncodeWithMaxLength(stream, msg); err != nil {
			return nil, err
		}
	}

	// Close stream for writing.
	if err := stream.Close(); err != nil {
		return nil, err
	}
	// Delay returning the stream for testing purposes
	if p.DelaySend {
		time.Sleep(1 * time.Second)
	}

	return stream, nil
}

// Started always returns true.
func (p *TestP2P) Started() bool {
	return true
}

// Peers returns the peer status.
func (p *TestP2P) Peers() *peers.Status {
	return p.peers
}

// FindPeersWithSubnet mocks the p2p func.
func (p *TestP2P) FindPeersWithSubnet(index uint64) (bool, error) {
	return false, nil
}

// RefreshENR mocks the p2p func.
func (p *TestP2P) RefreshENR() {
	return
}

// ForkDigest mocks the p2p func.
func (p *TestP2P) ForkDigest() ([4]byte, error) {
	return p.Digest, nil
}

// Metadata mocks the peer's metadata.
func (p *TestP2P) Metadata() *pb.MetaData {
	return proto.Clone(p.LocalMetadata).(*pb.MetaData)
}

// MetadataSeq mocks metadata sequence number.
func (p *TestP2P) MetadataSeq() uint64 {
	return p.LocalMetadata.SeqNumber
}

// AddPingMethod mocks the p2p func.
func (p *TestP2P) AddPingMethod(reqFunc func(ctx context.Context, id peer.ID) error) {
	// no-op
}

// InterceptPeerDial .
func (p *TestP2P) InterceptPeerDial(peer.ID) (allow bool) {
	return true
}

// InterceptAddrDial .
func (p *TestP2P) InterceptAddrDial(peer.ID, multiaddr.Multiaddr) (allow bool) {
	return true
}

// InterceptAccept .
func (p *TestP2P) InterceptAccept(n network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptSecured .
func (p *TestP2P) InterceptSecured(network.Direction, peer.ID, network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptUpgraded .
func (p *TestP2P) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}
