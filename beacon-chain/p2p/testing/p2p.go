package testing

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	bhost "github.com/libp2p/go-libp2p-blankhost"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	peers "github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

// TopicMappings are the protocol ids for the different types of requests.
var TopicMappings = map[reflect.Type]string{
	reflect.TypeOf(&pb.Status{}):                     "/eth2/beacon_chain/req/status/1",
	reflect.TypeOf(new(uint64)):                      "/eth2/beacon_chain/req/goodbye/1",
	reflect.TypeOf(&pb.BeaconBlocksByRangeRequest{}): "/eth2/beacon_chain/req/beacon_blocks_by_range/1",
	reflect.TypeOf([][32]byte{}):                     "/eth2/beacon_chain/req/beacon_blocks_by_root/1",
}

// TestP2P represents a p2p implementation that can be used for testing.
type TestP2P struct {
	t               *testing.T
	Host            host.Host
	pubsub          *pubsub.PubSub
	BroadcastCalled bool
	DelaySend       bool
	peers           *peers.Status
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
		Host:   h,
		pubsub: ps,
		peers:  peers.NewStatus(5 /* maxBadResponses */),
	}
}

// Connect two test peers together.
func (p *TestP2P) Connect(b *TestP2P) {
	if err := connect(p.Host, b.Host); err != nil {
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
	if err := connect(h, p.Host); err != nil {
		p.t.Fatalf("Failed to connect two peers for RPC: %v", err)
	}
	s, err := h.NewStream(context.Background(), p.Host.ID(), protocol.ID(topic+p.Encoding().ProtocolSuffix()))
	if err != nil {
		p.t.Fatalf("Failed to open stream %v", err)
	}
	defer s.Close()

	n, err := p.Encoding().EncodeWithLength(s, msg)
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
	if err := connect(h, p.Host); err != nil {
		p.t.Fatalf("Failed to connect two peers for RPC: %v", err)
	}

	// PubSub requires some delay after connecting for the (*PubSub).processLoop method to
	// pick up the newly connected peer.
	time.Sleep(time.Millisecond * 100)

	buf := new(bytes.Buffer)
	if _, err := p.Encoding().Encode(buf, msg); err != nil {
		p.t.Fatalf("Failed to encode message: %v", err)
	}

	if err := ps.Publish(topic+p.Encoding().ProtocolSuffix(), buf.Bytes()); err != nil {
		p.t.Fatalf("Failed to publish message; %v", err)
	}
}

// Broadcast a message.
func (p *TestP2P) Broadcast(ctx context.Context, msg proto.Message) error {
	p.BroadcastCalled = true
	return nil
}

// SetStreamHandler for RPC.
func (p *TestP2P) SetStreamHandler(topic string, handler network.StreamHandler) {
	p.Host.SetStreamHandler(protocol.ID(topic), handler)
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
	return p.Host.Network().ClosePeer(pid)
}

// PeerID returns the Peer ID of the local peer.
func (p *TestP2P) PeerID() peer.ID {
	return p.Host.ID()
}

// AddConnectionHandler handles the connection with a newly connected peer.
func (p *TestP2P) AddConnectionHandler(f func(ctx context.Context, id peer.ID) error) {
	p.Host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				p.peers.Add(conn.RemotePeer(), conn.RemoteMultiaddr(), conn.Stat().Direction)
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
	p.Host.Network().Notify(&network.NotifyBundle{
		DisconnectedF: func(net network.Network, conn network.Conn) {
			// Must be handled in a goroutine as this callback cannot be blocking.
			go func() {
				p.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnecting)
				f(context.Background(), conn.RemotePeer())
				p.peers.SetConnectionState(conn.RemotePeer(), peers.PeerDisconnected)
			}()
		},
	})
}

// Send a message to a specific peer.
func (p *TestP2P) Send(ctx context.Context, msg interface{}, pid peer.ID) (network.Stream, error) {
	protocol := TopicMappings[reflect.TypeOf(msg)]
	if protocol == "" {
		return nil, fmt.Errorf("protocol doesnt exist for proto message: %v", msg)
	}
	stream, err := p.Host.NewStream(ctx, pid, core.ProtocolID(protocol+p.Encoding().ProtocolSuffix()))
	if err != nil {
		return nil, err
	}

	if _, err := p.Encoding().EncodeWithLength(stream, msg); err != nil {
		return nil, err
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
