package testing

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	bhost "github.com/libp2p/go-libp2p-blankhost"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	pstoremem "github.com/libp2p/go-libp2p-peerstore/pstoremem"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarm "github.com/libp2p/go-libp2p-swarm"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	tnet "github.com/libp2p/go-libp2p-testing/net"
	"github.com/libp2p/go-tcp-transport"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// TestP2P represents a p2p implementation that can be used for testing.
type TestP2P struct {
	t               *testing.T
	Host            host.Host
	pubsub          *pubsub.PubSub
	BroadcastCalled bool
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
	}
}

// NewTestP2P initializes a new p2p test service.
func NewTestP2PWithKey(t *testing.T, privkey crypto.PrivKey) *TestP2P {
	ctx := context.Background()
	h := bhost.NewBlankHost(GenSwarmWithKey(t, ctx, privkey))
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
	}
}

// GenSwarmwithKey generates a new test swarm. Modified from
// "github.com/libp2p/go-libp2p-swarm/testing" for
// internal testing
func GenSwarmWithKey(t *testing.T, ctx context.Context, privKey crypto.PrivKey) *swarm.Swarm {
	var err error
	p := tnet.PeerNetParams{}
	p.Addr = tnet.ZeroLocalTCPAddress
	p.PrivKey = privKey
	p.PubKey = privKey.GetPublic()
	p.ID, err = peer.IDFromPublicKey(privKey.GetPublic())
	if err != nil {
		t.Fatal(err)
	}

	ps := pstoremem.NewPeerstore()
	ps.AddPubKey(p.ID, p.PubKey)
	ps.AddPrivKey(p.ID, p.PrivKey)
	s := swarm.NewSwarm(ctx, p.ID, ps, metrics.NewBandwidthCounter())

	tcpTransport := tcp.NewTCPTransport(swarmt.GenUpgrader(s))
	tcpTransport.DisableReuseport = false

	if err := s.AddTransport(tcpTransport); err != nil {
		t.Fatal(err)
	}

	if err := s.Listen(p.Addr); err != nil {
		t.Fatal(err)
	}

	s.Peerstore().AddAddrs(p.ID, s.ListenAddresses(), peerstore.PermanentAddrTTL)

	return s
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

	n, err := p.Encoding().Encode(s, msg)
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
func (p *TestP2P) Broadcast(msg proto.Message) error {
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

// AddHandshake to the peer handshake records.
func (p *TestP2P) AddHandshake(pid peer.ID, hello *pb.Hello) {
	// TODO(3147): add this.
}
