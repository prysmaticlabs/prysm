package debug

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockP2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
)

func TestDebugServer_GetPeer(t *testing.T) {
	peersProvider := &mockP2p.MockPeersProvider{}
	mP2P := mockP2p.NewTestP2P(t)
	ds := &Server{
		PeersFetcher: peersProvider,
		PeerManager:  &mockP2p.MockPeerManager{BHost: mP2P.BHost},
	}
	firstPeer := peersProvider.Peers().All()[0]

	res, err := ds.GetPeer(context.Background(), &ethpb.PeerRequest{PeerId: firstPeer.String()})
	if err != nil {
		t.Fatal(err)
	}
	if res.PeerId != firstPeer.String() {
		t.Fatalf("Expected peer id to be %s, but received: %s", firstPeer.String(), res.PeerId)
	}

	if int(res.Direction) != int(ethpb.PeerDirection_INBOUND) {
		t.Errorf("Expected 1st peer to be an inbound (%d) connection, received %d", ethpb.PeerDirection_INBOUND, res.Direction)
	}
	if res.ConnectionState != ethpb.ConnectionState_CONNECTED {
		t.Errorf("Expected peer to be connected received %s", res.ConnectionState.String())
	}
}

func TestDebugServer_ListPeers(t *testing.T) {
	peersProvider := &mockP2p.MockPeersProvider{}
	mP2P := mockP2p.NewTestP2P(t)
	ds := &Server{
		PeersFetcher: peersProvider,
		PeerManager:  &mockP2p.MockPeerManager{BHost: mP2P.BHost},
	}

	res, err := ds.ListPeers(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Responses) != 2 {
		t.Fatalf("Expected 2 peers, received %d: %v", len(res.Responses), res.Responses)
	}

	if int(res.Responses[0].Direction) != int(ethpb.PeerDirection_INBOUND) {
		t.Errorf("Expected 1st peer to be an inbound (%d) connection, received %d", ethpb.PeerDirection_INBOUND, res.Responses[0].Direction)
	}
	if len(res.Responses[0].ListeningAddresses) == 0 {
		t.Errorf("Expected 1st peer to have a multiaddress, instead they have no addresses")
	}
	if res.Responses[1].Direction != ethpb.PeerDirection_OUTBOUND {
		t.Errorf("Expected 2st peer to be an outbound (%d) connection, received %d", ethpb.PeerDirection_OUTBOUND, res.Responses[0].Direction)
	}
	if len(res.Responses[1].ListeningAddresses) == 0 {
		t.Errorf("Expected 2nd peer to have a multiaddress, instead they have no addresses")
	}
}
