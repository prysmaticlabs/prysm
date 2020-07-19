package node

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	mockP2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/shared/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func TestNodeServer_GetSyncStatus(t *testing.T) {
	mSync := &mockSync.Sync{IsSyncing: false}
	ns := &Server{
		SyncChecker: mSync,
	}
	res, err := ns.GetSyncStatus(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, false, res.Syncing)
	ns.SyncChecker = &mockSync.Sync{IsSyncing: true}
	res, err = ns.GetSyncStatus(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, true, res.Syncing)
}

func TestNodeServer_GetGenesis(t *testing.T) {
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	addr := common.Address{1, 2, 3}
	require.NoError(t, db.SaveDepositContractAddress(ctx, addr))
	st := testutil.NewBeaconState()
	genValRoot := bytesutil.ToBytes32([]byte("I am root"))
	ns := &Server{
		BeaconDB:           db,
		GenesisTimeFetcher: &mock.ChainService{},
		GenesisFetcher: &mock.ChainService{
			State:          st,
			ValidatorsRoot: genValRoot,
		},
	}
	res, err := ns.GetGenesis(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.DeepEqual(t, addr.Bytes(), res.DepositContractAddress)
	pUnix, err := ptypes.TimestampProto(time.Unix(0, 0))
	require.NoError(t, err)
	assert.Equal(t, true, res.GenesisTime.Equal(pUnix))
	assert.DeepEqual(t, genValRoot[:], res.GenesisValidatorsRoot)

	ns.GenesisTimeFetcher = &mock.ChainService{Genesis: time.Unix(10, 0)}
	res, err = ns.GetGenesis(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	pUnix, err = ptypes.TimestampProto(time.Unix(10, 0))
	require.NoError(t, err)
	assert.Equal(t, true, res.GenesisTime.Equal(pUnix))
}

func TestNodeServer_GetVersion(t *testing.T) {
	v := version.GetVersion()
	ns := &Server{}
	res, err := ns.GetVersion(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, v, res.Version)
}

func TestNodeServer_GetImplementedServices(t *testing.T) {
	server := grpc.NewServer()
	ns := &Server{
		Server: server,
	}
	ethpb.RegisterNodeServer(server, ns)
	reflection.Register(server)

	res, err := ns.ListImplementedServices(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	// We verify the services include the node service + the registered reflection service.
	assert.Equal(t, 2, len(res.Services))
}

func TestNodeServer_GetHost(t *testing.T) {
	server := grpc.NewServer()
	peersProvider := &mockP2p.MockPeersProvider{}
	mP2P := mockP2p.NewTestP2P(t)
	key, err := crypto.GenerateKey()
	db, err := enode.OpenDB("")
	require.NoError(t, err)
	lNode := enode.NewLocalNode(db, key)
	record := lNode.Node().Record()
	stringENR, err := p2p.SerializeENR(record)
	require.NoError(t, err)
	ns := &Server{
		PeerManager:  &mockP2p.MockPeerManager{BHost: mP2P.BHost, Enr: record, PID: mP2P.BHost.ID()},
		PeersFetcher: peersProvider,
	}
	ethpb.RegisterNodeServer(server, ns)
	reflection.Register(server)
	h, err := ns.GetHost(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, mP2P.PeerID().String(), h.PeerId)
	assert.Equal(t, stringENR, h.Enr)
}

func TestNodeServer_GetPeer(t *testing.T) {
	server := grpc.NewServer()
	peersProvider := &mockP2p.MockPeersProvider{}
	ns := &Server{
		PeersFetcher: peersProvider,
	}
	ethpb.RegisterNodeServer(server, ns)
	reflection.Register(server)
	firstPeer := peersProvider.Peers().All()[0]

	res, err := ns.GetPeer(context.Background(), &ethpb.PeerRequest{PeerId: firstPeer.String()})
	require.NoError(t, err)
	assert.Equal(t, firstPeer.String(), res.PeerId, "Unexpected peer ID")
	assert.Equal(t, int(ethpb.PeerDirection_INBOUND), int(res.Direction), "Expected 1st peer to be an inbound connection")
	assert.Equal(t, ethpb.ConnectionState_CONNECTED, res.ConnectionState, "Expected peer to be connected")
}

func TestNodeServer_ListPeers(t *testing.T) {
	server := grpc.NewServer()
	peersProvider := &mockP2p.MockPeersProvider{}
	ns := &Server{
		PeersFetcher: peersProvider,
	}
	ethpb.RegisterNodeServer(server, ns)
	reflection.Register(server)

	res, err := ns.ListPeers(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(res.Peers))
	assert.Equal(t, int(ethpb.PeerDirection_INBOUND), int(res.Peers[0].Direction))
	assert.Equal(t, ethpb.PeerDirection_OUTBOUND, res.Peers[1].Direction)
}
