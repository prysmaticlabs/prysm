package debug

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	mockP2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
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
	require.NoError(t, err)
	require.Equal(t, firstPeer.String(), res.PeerId, "Unexpected peer ID")

	assert.Equal(t, int(ethpb.PeerDirection_INBOUND), int(res.Direction), "Expected 1st peer to be an inbound connection")
	assert.Equal(t, ethpb.ConnectionState_CONNECTED, res.ConnectionState, "Expected peer to be connected")
}

func TestDebugServer_ListPeers(t *testing.T) {
	peersProvider := &mockP2p.MockPeersProvider{}
	mP2P := mockP2p.NewTestP2P(t)
	ds := &Server{
		PeersFetcher: peersProvider,
		PeerManager:  &mockP2p.MockPeerManager{BHost: mP2P.BHost},
	}

	res, err := ds.ListPeers(context.Background(), &empty.Empty{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(res.Responses))

	assert.Equal(t, int(ethpb.PeerDirection_INBOUND), int(res.Responses[0].Direction), "Expected 1st peer to be an inbound")
	if len(res.Responses[0].ListeningAddresses) == 0 {
		t.Errorf("Expected 1st peer to have a multiaddress, instead they have no addresses")
	}
	assert.Equal(t, ethpb.PeerDirection_OUTBOUND, res.Responses[1].Direction, "Expected 2st peer to be an outbound connection")
	if len(res.Responses[1].ListeningAddresses) == 0 {
		t.Errorf("Expected 2nd peer to have a multiaddress, instead they have no addresses")
	}
}
