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

type fakeP2P struct {

}

func NewFuzzTestP2P() *fakeP2P {
	return &fakeP2P{}
}

func (f *fakeP2P) Encoding() encoder.NetworkEncoding {
	return &encoder.SszNetworkEncoder{}
}

func (p *fakeP2P) AddConnectionHandler(f func(ctx context.Context, id peer.ID) error) {

}

func (p *fakeP2P) AddDisconnectionHandler(f func(ctx context.Context, id peer.ID) error) {
}

func (p *fakeP2P) AddPingMethod(reqFunc func(ctx context.Context, id peer.ID) error) {

}

func (p *fakeP2P) PeerID() peer.ID {
	return peer.ID("fake")
}

// ENR returns the enr of the local peer.
func (p *fakeP2P) ENR() *enr.Record {
	return new(enr.Record)
}

// FindPeersWithSubnet mocks the p2p func.
func (p *fakeP2P) FindPeersWithSubnet(index uint64) (bool, error) {
	return false, nil
}

// RefreshENR mocks the p2p func.
func (p *fakeP2P) RefreshENR() {
	return
}

func (p *fakeP2P) LeaveTopic(topic string) error {
	return nil

}

func (p *fakeP2P) Metadata() *pb.MetaData {
	return nil
}

func (p *fakeP2P) Peers() *peers.Status {
	return nil
}

func (p *fakeP2P) PublishToTopic(ctx context.Context, topic string, data []byte, opts ...pubsub.PubOpt) error {
	return nil
}

func (p *fakeP2P) Send(ctx context.Context, msg interface{}, topic string, pid peer.ID) (network.Stream, error) {
	return nil, nil
}

func (p *fakeP2P) PubSub() *pubsub.PubSub {
	return nil
}

func (p *fakeP2P) MetadataSeq() uint64 {
	return 0
}

func (p *fakeP2P) SetStreamHandler(topic string, handler network.StreamHandler) {

}

func (p *fakeP2P) SubscribeToTopic(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error) {
	return nil, nil
}

func (p *fakeP2P) JoinTopic(topic string, opts ...pubsub.TopicOpt) (*pubsub.Topic, error) {
 return nil, nil
}

func (p *fakeP2P) Host() host.Host {
	return nil
}

func (p *fakeP2P) Disconnect(pid peer.ID) error {
	return nil
}

// Broadcast a message.
func (p *fakeP2P) Broadcast(ctx context.Context, msg proto.Message) error {
	return nil
}

// BroadcastAttestation broadcasts an attestation.
func (p *fakeP2P) BroadcastAttestation(ctx context.Context, subnet uint64, att *ethpb.Attestation) error {
	return nil
}

// InterceptPeerDial .
func (p *fakeP2P) InterceptPeerDial(peer.ID) (allow bool) {
	return true
}

// InterceptAddrDial .
func (p *fakeP2P) InterceptAddrDial(peer.ID, multiaddr.Multiaddr) (allow bool) {
	return true
}

// InterceptAccept .
func (p *fakeP2P) InterceptAccept(n network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptSecured .
func (p *fakeP2P) InterceptSecured(network.Direction, peer.ID, network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptUpgraded .
func (p *fakeP2P) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}
