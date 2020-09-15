package testing

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// FakeP2P stack
type FakeP2P struct {
}

// NewFuzzTestP2P - Create a new fake p2p stack.
func NewFuzzTestP2P() *FakeP2P {
	return &FakeP2P{}
}

// Encoding -- fake.
func (p *FakeP2P) Encoding() encoder.NetworkEncoding {
	return &encoder.SszNetworkEncoder{}
}

// AddConnectionHandler -- fake.
func (p *FakeP2P) AddConnectionHandler(f func(ctx context.Context, id peer.ID) error) {

}

// AddDisconnectionHandler -- fake.
func (p *FakeP2P) AddDisconnectionHandler(f func(ctx context.Context, id peer.ID) error) {
}

// AddPingMethod -- fake.
func (p *FakeP2P) AddPingMethod(reqFunc func(ctx context.Context, id peer.ID) error) {

}

// PeerID -- fake.
func (p *FakeP2P) PeerID() peer.ID {
	return peer.ID("fake")
}

// ENR returns the enr of the local peer.
func (p *FakeP2P) ENR() *enr.Record {
	return new(enr.Record)
}

// FindPeersWithSubnet mocks the p2p func.
func (p *FakeP2P) FindPeersWithSubnet(ctx context.Context, index uint64) (bool, error) {
	return false, nil
}

// RefreshENR mocks the p2p func.
func (p *FakeP2P) RefreshENR() {
	return
}

// LeaveTopic -- fake.
func (p *FakeP2P) LeaveTopic(topic string) error {
	return nil

}

// Metadata -- fake.
func (p *FakeP2P) Metadata() *pb.MetaData {
	return nil
}

// Peers -- fake.
func (p *FakeP2P) Peers() *peers.Status {
	return nil
}

// PublishToTopic -- fake.
func (p *FakeP2P) PublishToTopic(ctx context.Context, topic string, data []byte, opts ...pubsub.PubOpt) error {
	return nil
}

// Send -- fake.
func (p *FakeP2P) Send(ctx context.Context, msg interface{}, topic string, pid peer.ID) (network.Stream, error) {
	return nil, nil
}

// PubSub -- fake.
func (p *FakeP2P) PubSub() *pubsub.PubSub {
	return nil
}

// MetadataSeq -- fake.
func (p *FakeP2P) MetadataSeq() uint64 {
	return 0
}

// SetStreamHandler -- fake.
func (p *FakeP2P) SetStreamHandler(topic string, handler network.StreamHandler) {

}

// SubscribeToTopic -- fake.
func (p *FakeP2P) SubscribeToTopic(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error) {
	return nil, nil
}

// JoinTopic -- fake.
func (p *FakeP2P) JoinTopic(topic string, opts ...pubsub.TopicOpt) (*pubsub.Topic, error) {
	return nil, nil
}

// Host -- fake.
func (p *FakeP2P) Host() host.Host {
	return nil
}

// Disconnect -- fake.
func (p *FakeP2P) Disconnect(pid peer.ID) error {
	return nil
}

// Broadcast -- fake.
func (p *FakeP2P) Broadcast(ctx context.Context, msg proto.Message) error {
	return nil
}

// BroadcastAttestation -- fake.
func (p *FakeP2P) BroadcastAttestation(ctx context.Context, subnet uint64, att *ethpb.Attestation) error {
	return nil
}

// InterceptPeerDial -- fake.
func (p *FakeP2P) InterceptPeerDial(peer.ID) (allow bool) {
	return true
}

// InterceptAddrDial -- fake.
func (p *FakeP2P) InterceptAddrDial(peer.ID, multiaddr.Multiaddr) (allow bool) {
	return true
}

// InterceptAccept -- fake.
func (p *FakeP2P) InterceptAccept(n network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptSecured -- fake.
func (p *FakeP2P) InterceptSecured(network.Direction, peer.ID, network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptUpgraded -- fake.
func (p *FakeP2P) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}
