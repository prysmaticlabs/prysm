package node

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	libp2ptest "github.com/libp2p/go-libp2p-peerstore/test"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/go-bitfield"
	grpcutil "github.com/prysmaticlabs/prysm/v3/api/grpc"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/peers"
	mockp2p "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	syncmock "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/wrapper"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type dummyIdentity enode.ID

func (_ dummyIdentity) Verify(_ *enr.Record, _ []byte) error { return nil }
func (id dummyIdentity) NodeAddr(_ *enr.Record) []byte       { return id[:] }

func TestGetVersion(t *testing.T) {
	semVer := version.SemanticVersion()
	os := runtime.GOOS
	arch := runtime.GOARCH
	res, err := (&Server{}).GetVersion(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	v := res.Data.Version
	assert.Equal(t, true, strings.Contains(v, semVer))
	assert.Equal(t, true, strings.Contains(v, os))
	assert.Equal(t, true, strings.Contains(v, arch))
}

func TestGetHealth(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &grpcruntime.ServerTransportStream{})
	checker := &syncmock.Sync{}
	s := &Server{
		SyncChecker: checker,
	}

	_, err := s.GetHealth(ctx, &emptypb.Empty{})
	require.ErrorContains(t, "Node not initialized or having issues", err)
	checker.IsInitialized = true
	_, err = s.GetHealth(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	stream, ok := grpc.ServerTransportStreamFromContext(ctx).(*grpcruntime.ServerTransportStream)
	require.Equal(t, true, ok, "type assertion failed")
	assert.Equal(t, stream.Header()[strings.ToLower(grpcutil.HttpCodeMetadataKey)][0], strconv.Itoa(http.StatusPartialContent))
	checker.IsSynced = true
	_, err = s.GetHealth(ctx, &emptypb.Empty{})
	require.NoError(t, err)
}

func TestGetIdentity(t *testing.T) {
	ctx := context.Background()
	p2pAddr, err := ma.NewMultiaddr("/ip4/7.7.7.7/udp/30303")
	require.NoError(t, err)
	discAddr1, err := ma.NewMultiaddr("/ip4/7.7.7.7/udp/30303/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	require.NoError(t, err)
	discAddr2, err := ma.NewMultiaddr("/ip6/1:2:3:4:5:6:7:8/udp/20202/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	require.NoError(t, err)
	enrRecord := &enr.Record{}
	err = enrRecord.SetSig(dummyIdentity{1}, []byte{42})
	require.NoError(t, err)
	enrRecord.Set(enr.IPv4{7, 7, 7, 7})
	err = enrRecord.SetSig(dummyIdentity{}, []byte{})
	require.NoError(t, err)
	attnets := bitfield.NewBitvector64()
	attnets.SetBitAt(1, true)
	metadataProvider := &mockp2p.MockMetadataProvider{Data: wrapper.WrappedMetadataV0(&pb.MetaDataV0{SeqNumber: 1, Attnets: attnets})}

	t.Run("OK", func(t *testing.T) {
		peerManager := &mockp2p.MockPeerManager{
			Enr:           enrRecord,
			PID:           "foo",
			BHost:         &mockp2p.MockHost{Addresses: []ma.Multiaddr{p2pAddr}},
			DiscoveryAddr: []ma.Multiaddr{discAddr1, discAddr2},
		}
		s := &Server{
			PeerManager:      peerManager,
			MetadataProvider: metadataProvider,
		}

		resp, err := s.GetIdentity(ctx, &emptypb.Empty{})
		require.NoError(t, err)
		expectedID := peer.ID("foo").Pretty()
		assert.Equal(t, expectedID, resp.Data.PeerId)
		expectedEnr, err := p2p.SerializeENR(enrRecord)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprint("enr:", expectedEnr), resp.Data.Enr)
		require.Equal(t, 1, len(resp.Data.P2PAddresses))
		assert.Equal(t, p2pAddr.String()+"/p2p/"+expectedID, resp.Data.P2PAddresses[0])
		require.Equal(t, 2, len(resp.Data.DiscoveryAddresses))
		ipv4Found, ipv6Found := false, false
		for _, address := range resp.Data.DiscoveryAddresses {
			if address == discAddr1.String() {
				ipv4Found = true
			} else if address == discAddr2.String() {
				ipv6Found = true
			}
		}
		assert.Equal(t, true, ipv4Found, "IPv4 discovery address not found")
		assert.Equal(t, true, ipv6Found, "IPv6 discovery address not found")
		assert.Equal(t, discAddr1.String(), resp.Data.DiscoveryAddresses[0])
		assert.Equal(t, discAddr2.String(), resp.Data.DiscoveryAddresses[1])
	})

	t.Run("ENR failure", func(t *testing.T) {
		peerManager := &mockp2p.MockPeerManager{
			Enr:           &enr.Record{},
			PID:           "foo",
			BHost:         &mockp2p.MockHost{Addresses: []ma.Multiaddr{p2pAddr}},
			DiscoveryAddr: []ma.Multiaddr{discAddr1, discAddr2},
		}
		s := &Server{
			PeerManager:      peerManager,
			MetadataProvider: metadataProvider,
		}

		_, err = s.GetIdentity(ctx, &emptypb.Empty{})
		assert.ErrorContains(t, "Could not obtain enr", err)
	})

	t.Run("Discovery addresses failure", func(t *testing.T) {
		peerManager := &mockp2p.MockPeerManager{
			Enr:               enrRecord,
			PID:               "foo",
			BHost:             &mockp2p.MockHost{Addresses: []ma.Multiaddr{p2pAddr}},
			DiscoveryAddr:     []ma.Multiaddr{discAddr1, discAddr2},
			FailDiscoveryAddr: true,
		}
		s := &Server{
			PeerManager:      peerManager,
			MetadataProvider: metadataProvider,
		}

		_, err = s.GetIdentity(ctx, &emptypb.Empty{})
		assert.ErrorContains(t, "Could not obtain discovery address", err)
	})
}

func TestSyncStatus(t *testing.T) {
	currentSlot := new(types.Slot)
	*currentSlot = 110
	state, err := util.NewBeaconState()
	require.NoError(t, err)
	err = state.SetSlot(100)
	require.NoError(t, err)
	chainService := &mock.ChainService{Slot: currentSlot, State: state, Optimistic: true}
	syncChecker := &syncmock.Sync{}
	syncChecker.IsSyncing = true

	s := &Server{
		HeadFetcher:           chainService,
		GenesisTimeFetcher:    chainService,
		OptimisticModeFetcher: chainService,
		SyncChecker:           syncChecker,
	}
	resp, err := s.GetSyncStatus(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, types.Slot(100), resp.Data.HeadSlot)
	assert.Equal(t, types.Slot(10), resp.Data.SyncDistance)
	assert.Equal(t, true, resp.Data.IsSyncing)
	assert.Equal(t, true, resp.Data.IsOptimistic)
}

func TestGetPeer(t *testing.T) {
	const rawId = "16Uiu2HAkvyYtoQXZNTsthjgLHjEnv7kvwzEmjvsJjWXpbhtqpSUN"
	ctx := context.Background()
	decodedId, err := peer.Decode(rawId)
	require.NoError(t, err)
	enrRecord := &enr.Record{}
	err = enrRecord.SetSig(dummyIdentity{1}, []byte{42})
	require.NoError(t, err)
	enrRecord.Set(enr.IPv4{7, 7, 7, 7})
	err = enrRecord.SetSig(dummyIdentity{}, []byte{})
	require.NoError(t, err)
	const p2pAddr = "/ip4/7.7.7.7/udp/30303/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N"
	p2pMultiAddr, err := ma.NewMultiaddr(p2pAddr)
	require.NoError(t, err)
	peerFetcher := &mockp2p.MockPeersProvider{}
	s := Server{PeersFetcher: peerFetcher}
	peerFetcher.Peers().Add(enrRecord, decodedId, p2pMultiAddr, network.DirInbound)

	t.Run("OK", func(t *testing.T) {
		resp, err := s.GetPeer(ctx, &ethpb.PeerRequest{PeerId: rawId})
		require.NoError(t, err)
		assert.Equal(t, rawId, resp.Data.PeerId)
		assert.Equal(t, p2pAddr, resp.Data.LastSeenP2PAddress)
		assert.Equal(t, "enr:yoABgmlwhAcHBwc=", resp.Data.Enr)
		assert.Equal(t, ethpb.ConnectionState_DISCONNECTED, resp.Data.State)
		assert.Equal(t, ethpb.PeerDirection_INBOUND, resp.Data.Direction)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		_, err = s.GetPeer(ctx, &ethpb.PeerRequest{PeerId: "foo"})
		assert.ErrorContains(t, "Invalid peer ID", err)
	})

	t.Run("Peer not found", func(t *testing.T) {
		generatedId := "16Uiu2HAmQqFdEcHbSmQTQuLoAhnMUrgoWoraKK4cUJT6FuuqHqTU"
		_, err = s.GetPeer(ctx, &ethpb.PeerRequest{PeerId: generatedId})
		assert.ErrorContains(t, "Peer not found", err)
	})
}

func TestListPeers(t *testing.T) {
	ids := libp2ptest.GeneratePeerIDs(9)
	peerFetcher := &mockp2p.MockPeersProvider{}
	peerFetcher.ClearPeers()
	peerStatus := peerFetcher.Peers()

	for i, id := range ids {
		// Make last peer undiscovered
		if i == len(ids)-1 {
			peerStatus.Add(nil, id, nil, network.DirUnknown)
		} else {
			enrRecord := &enr.Record{}
			err := enrRecord.SetSig(dummyIdentity{1}, []byte{42})
			require.NoError(t, err)
			enrRecord.Set(enr.IPv4{127, 0, 0, byte(i)})
			err = enrRecord.SetSig(dummyIdentity{}, []byte{})
			require.NoError(t, err)
			var p2pAddr = "/ip4/127.0.0." + strconv.Itoa(i) + "/udp/30303/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N"
			p2pMultiAddr, err := ma.NewMultiaddr(p2pAddr)
			require.NoError(t, err)

			var direction network.Direction
			if i%2 == 0 {
				direction = network.DirInbound
			} else {
				direction = network.DirOutbound
			}
			peerStatus.Add(enrRecord, id, p2pMultiAddr, direction)

			switch i {
			case 0, 1:
				peerStatus.SetConnectionState(id, peers.PeerConnecting)
			case 2, 3:
				peerStatus.SetConnectionState(id, peers.PeerConnected)
			case 4, 5:
				peerStatus.SetConnectionState(id, peers.PeerDisconnecting)
			case 6, 7:
				peerStatus.SetConnectionState(id, peers.PeerDisconnected)
			default:
				t.Fatalf("Failed to set connection state for peer")
			}
		}
	}

	s := Server{PeersFetcher: peerFetcher}

	t.Run("Peer data OK", func(t *testing.T) {
		// We will check the first peer from the list.
		expectedId := ids[0]

		resp, err := s.ListPeers(context.Background(), &ethpb.PeersRequest{
			State:     []ethpb.ConnectionState{ethpb.ConnectionState_CONNECTING},
			Direction: []ethpb.PeerDirection{ethpb.PeerDirection_INBOUND},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Data))
		returnedPeer := resp.Data[0]
		assert.Equal(t, expectedId.Pretty(), returnedPeer.PeerId)
		expectedEnr, err := peerStatus.ENR(expectedId)
		require.NoError(t, err)
		serializedEnr, err := p2p.SerializeENR(expectedEnr)
		require.NoError(t, err)
		assert.Equal(t, "enr:"+serializedEnr, returnedPeer.Enr)
		expectedP2PAddr, err := peerStatus.Address(expectedId)
		require.NoError(t, err)
		assert.Equal(t, expectedP2PAddr.String(), returnedPeer.LastSeenP2PAddress)
		assert.Equal(t, ethpb.ConnectionState_CONNECTING, returnedPeer.State)
		assert.Equal(t, ethpb.PeerDirection_INBOUND, returnedPeer.Direction)
	})

	filterTests := []struct {
		name       string
		states     []ethpb.ConnectionState
		directions []ethpb.PeerDirection
		wantIds    []peer.ID
	}{
		{
			name:       "No filters - return all peers",
			states:     []ethpb.ConnectionState{},
			directions: []ethpb.PeerDirection{},
			wantIds:    ids[:len(ids)-1], // Excluding last peer as it is not connected.
		},
		{
			name:       "State filter empty - return peers for all states",
			states:     []ethpb.ConnectionState{},
			directions: []ethpb.PeerDirection{ethpb.PeerDirection_INBOUND},
			wantIds:    []peer.ID{ids[0], ids[2], ids[4], ids[6]},
		},
		{
			name:       "Direction filter empty - return peers for all directions",
			states:     []ethpb.ConnectionState{ethpb.ConnectionState_CONNECTED},
			directions: []ethpb.PeerDirection{},
			wantIds:    []peer.ID{ids[2], ids[3]},
		},
		{
			name:       "One state and direction",
			states:     []ethpb.ConnectionState{ethpb.ConnectionState_DISCONNECTED},
			directions: []ethpb.PeerDirection{ethpb.PeerDirection_INBOUND},
			wantIds:    []peer.ID{ids[6]},
		},
		{
			name:       "Multiple states and directions",
			states:     []ethpb.ConnectionState{ethpb.ConnectionState_CONNECTING, ethpb.ConnectionState_DISCONNECTING},
			directions: []ethpb.PeerDirection{ethpb.PeerDirection_INBOUND, ethpb.PeerDirection_OUTBOUND},
			wantIds:    []peer.ID{ids[0], ids[1], ids[4], ids[5]},
		},
		{
			name:       "Unknown filter is ignored",
			states:     []ethpb.ConnectionState{ethpb.ConnectionState_CONNECTED, 99},
			directions: []ethpb.PeerDirection{ethpb.PeerDirection_OUTBOUND, 99},
			wantIds:    []peer.ID{ids[3]},
		},
		{
			name:       "Only unknown filters - return all peers",
			states:     []ethpb.ConnectionState{99},
			directions: []ethpb.PeerDirection{99},
			wantIds:    ids[:len(ids)-1], // Excluding last peer as it is not connected.
		},
	}
	for _, tt := range filterTests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := s.ListPeers(context.Background(), &ethpb.PeersRequest{
				State:     tt.states,
				Direction: tt.directions,
			})
			require.NoError(t, err)
			assert.Equal(t, len(tt.wantIds), len(resp.Data), "Wrong number of peers returned")
			for _, id := range tt.wantIds {
				expectedId := id.Pretty()
				found := false
				for _, returnedPeer := range resp.Data {
					if returnedPeer.PeerId == expectedId {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected ID '" + expectedId + "' not found")
				}
			}
		})
	}
}

func TestListPeers_NoPeersReturnsEmptyArray(t *testing.T) {
	peerFetcher := &mockp2p.MockPeersProvider{}
	peerFetcher.ClearPeers()
	s := Server{PeersFetcher: peerFetcher}

	resp, err := s.ListPeers(context.Background(), &ethpb.PeersRequest{
		State: []ethpb.ConnectionState{ethpb.ConnectionState_CONNECTED},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	assert.Equal(t, 0, len(resp.Data))
}

func TestPeerCount(t *testing.T) {
	ids := libp2ptest.GeneratePeerIDs(10)
	peerFetcher := &mockp2p.MockPeersProvider{}
	peerFetcher.ClearPeers()
	peerStatus := peerFetcher.Peers()

	for i, id := range ids {
		enrRecord := &enr.Record{}
		err := enrRecord.SetSig(dummyIdentity{1}, []byte{42})
		require.NoError(t, err)
		enrRecord.Set(enr.IPv4{127, 0, 0, byte(i)})
		err = enrRecord.SetSig(dummyIdentity{}, []byte{})
		require.NoError(t, err)
		var p2pAddr = "/ip4/127.0.0." + strconv.Itoa(i) + "/udp/30303/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N"
		p2pMultiAddr, err := ma.NewMultiaddr(p2pAddr)
		require.NoError(t, err)

		var direction network.Direction
		if i%2 == 0 {
			direction = network.DirInbound
		} else {
			direction = network.DirOutbound
		}
		peerStatus.Add(enrRecord, id, p2pMultiAddr, direction)

		switch i {
		case 0:
			peerStatus.SetConnectionState(id, peers.PeerConnecting)
		case 1, 2:
			peerStatus.SetConnectionState(id, peers.PeerConnected)
		case 3, 4, 5:
			peerStatus.SetConnectionState(id, peers.PeerDisconnecting)
		case 6, 7, 8, 9:
			peerStatus.SetConnectionState(id, peers.PeerDisconnected)
		default:
			t.Fatalf("Failed to set connection state for peer")
		}
	}

	s := Server{PeersFetcher: peerFetcher}
	resp, err := s.PeerCount(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, uint64(1), resp.Data.Connecting, "Wrong number of connecting peers")
	assert.Equal(t, uint64(2), resp.Data.Connected, "Wrong number of connected peers")
	assert.Equal(t, uint64(3), resp.Data.Disconnecting, "Wrong number of disconnecting peers")
	assert.Equal(t, uint64(4), resp.Data.Disconnected, "Wrong number of disconnected peers")
}

func BenchmarkListPeers(b *testing.B) {
	// We simulate having a lot of peers.
	ids := libp2ptest.GeneratePeerIDs(2000)
	peerFetcher := &mockp2p.MockPeersProvider{}

	for _, id := range ids {
		enrRecord := &enr.Record{}
		err := enrRecord.SetSig(dummyIdentity{1}, []byte{42})
		require.NoError(b, err)
		enrRecord.Set(enr.IPv4{7, 7, 7, 7})
		err = enrRecord.SetSig(dummyIdentity{}, []byte{})
		require.NoError(b, err)
		const p2pAddr = "/ip4/7.7.7.7/udp/30303/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N"
		p2pMultiAddr, err := ma.NewMultiaddr(p2pAddr)
		require.NoError(b, err)
		peerFetcher.Peers().Add(enrRecord, id, p2pMultiAddr, network.DirInbound)
	}

	s := Server{PeersFetcher: peerFetcher}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.ListPeers(context.Background(), &ethpb.PeersRequest{
			State:     []ethpb.ConnectionState{},
			Direction: []ethpb.PeerDirection{},
		})
		require.NoError(b, err)
	}
}
