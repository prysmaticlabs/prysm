package nodev1

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-peerstore/test"
	ma "github.com/multiformats/go-multiaddr"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	syncmock "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/version"
)

type dummyIdentity enode.ID

func (id dummyIdentity) Verify(_ *enr.Record, _ []byte) error { return nil }
func (id dummyIdentity) NodeAddr(_ *enr.Record) []byte        { return id[:] }

func TestGetVersion(t *testing.T) {
	semVer := version.GetSemanticVersion()
	os := runtime.GOOS
	arch := runtime.GOARCH
	res, err := (&Server{}).GetVersion(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	v := res.Data.Version
	assert.Equal(t, true, strings.Contains(v, semVer))
	assert.Equal(t, true, strings.Contains(v, os))
	assert.Equal(t, true, strings.Contains(v, arch))
}

func TestGetHealth(t *testing.T) {
	ctx := context.Background()
	checker := &syncmock.Sync{}
	s := &Server{
		SyncChecker: checker,
	}

	_, err := s.GetHealth(ctx, &ptypes.Empty{})
	require.ErrorContains(t, "Node not initialized or having issues", err)
	checker.IsInitialized = true
	_, err = s.GetHealth(ctx, &ptypes.Empty{})
	require.NoError(t, err)
	checker.IsInitialized = false
	checker.IsSyncing = true
	require.NoError(t, err)
}

func TestSyncStatus(t *testing.T) {
	currentSlot := new(uint64)
	*currentSlot = 110
	state := testutil.NewBeaconState()
	err := state.SetSlot(100)
	require.NoError(t, err)
	chainService := &mock.ChainService{Slot: currentSlot, State: state}

	s := &Server{
		HeadFetcher:        chainService,
		GenesisTimeFetcher: chainService,
	}
	resp, err := s.GetSyncStatus(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, uint64(100), resp.Data.HeadSlot)
	assert.Equal(t, uint64(10), resp.Data.SyncDistance)
}

func TestGetPeer(t *testing.T) {
	ctx := context.Background()
	decodedId, err := peer.Decode("16Uiu2HAkvyYtoQXZNTsthjgLHjEnv7kvwzEmjvsJjWXpbhtqpSUN")
	require.NoError(t, err)
	peerId := string(decodedId)
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
		resp, err := s.GetPeer(ctx, &ethpb.PeerRequest{PeerId: peerId})
		require.NoError(t, err)
		assert.Equal(t, peerId, resp.Data.PeerId)
		assert.Equal(t, p2pAddr, resp.Data.Address)
		assert.Equal(t, "enr:yoABgmlwhAcHBwc=", resp.Data.Enr)
		assert.Equal(t, ethpb.ConnectionState_DISCONNECTED, resp.Data.State)
		assert.Equal(t, ethpb.PeerDirection_INBOUND, resp.Data.Direction)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		_, err = s.GetPeer(ctx, &ethpb.PeerRequest{PeerId: "foo"})
		assert.ErrorContains(t, "Invalid peer ID: foo", err)
	})

	t.Run("Peer not found", func(t *testing.T) {
		generatedId := string(test.GeneratePeerIDs(1)[0])
		_, err = s.GetPeer(ctx, &ethpb.PeerRequest{PeerId: generatedId})
		assert.ErrorContains(t, "Peer not found", err)
	})
}
