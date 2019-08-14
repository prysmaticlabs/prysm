package testing

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	bhost "github.com/libp2p/go-libp2p-blankhost"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
)

var _ = p2p.Composite(&TestP2P{})

type TestP2P struct {
	t      *testing.T
	host   host.Host
	pubsub *pubsub.PubSub
}

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
		host:   h,
		pubsub: ps,
	}
}

// Connect two test peers together.
func (p *TestP2P) Connect(b *TestP2P) {
	if err := connect(p.host, b.host); err != nil {
		p.t.Fatal(err)
	}
}

func connect(a, b host.Host) error {
	pinfo := a.Peerstore().PeerInfo(a.ID())
	return b.Connect(context.Background(), pinfo)
}

// ReceiveRPC simulates an incoming RPC.
func (p *TestP2P) ReceiveRPC(topic string, msg proto.Message) {
	h := bhost.NewBlankHost(swarmt.GenSwarm(p.t, context.Background()))
	if err := connect(h, p.host); err != nil {
		p.t.Fatalf("Failed to connect two peers for RPC: %v", err)
	}
	s, err := h.NewStream(context.Background(), p.host.ID(), protocol.ID(topic+p.Encoding().ProtocolSuffix()))
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

// Broadcast a message.
func (p *TestP2P) Broadcast(msg proto.Message) {

}

// SetStreamHandler for RPC.
func (p *TestP2P) SetStreamHandler(topic string, handler network.StreamHandler) {
	p.host.SetStreamHandler(protocol.ID(topic), handler)
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
