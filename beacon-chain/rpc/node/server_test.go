package node

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockP2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
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
	if err != nil {
		t.Fatal(err)
	}
	if res.Syncing {
		t.Errorf("Wanted GetSyncStatus() = %v, received %v", false, res.Syncing)
	}
	ns.SyncChecker = &mockSync.Sync{IsSyncing: true}
	res, err = ns.GetSyncStatus(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Syncing {
		t.Errorf("Wanted GetSyncStatus() = %v, received %v", true, res.Syncing)
	}
}

func TestNodeServer_GetGenesis(t *testing.T) {
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	ctx := context.Background()
	addr := common.Address{1, 2, 3}
	if err := db.SaveDepositContractAddress(ctx, addr); err != nil {
		t.Fatal(err)
	}
	ns := &Server{
		BeaconDB:           db,
		GenesisTimeFetcher: &mock.ChainService{Genesis: time.Unix(0, 0)},
	}
	res, err := ns.GetGenesis(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(res.DepositContractAddress, addr.Bytes()) {
		t.Errorf("Wanted DepositContractAddress() = %#x, received %#x", addr.Bytes(), res.DepositContractAddress)
	}
	pUnix, err := ptypes.TimestampProto(time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !res.GenesisTime.Equal(pUnix) {
		t.Errorf("Wanted GenesisTime() = %v, received %v", pUnix, res.GenesisTime)
	}
}

func TestNodeServer_GetVersion(t *testing.T) {
	v := version.GetVersion()
	ns := &Server{}
	res, err := ns.GetVersion(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Version != v {
		t.Errorf("Wanted GetVersion() = %s, received %s", v, res.Version)
	}
}

func TestNodeServer_GetImplementedServices(t *testing.T) {
	server := grpc.NewServer()
	ns := &Server{
		Server: server,
	}
	ethpb.RegisterNodeServer(server, ns)
	reflection.Register(server)

	res, err := ns.ListImplementedServices(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	// We verify the services include the node service + the registered reflection service.
	if len(res.Services) != 2 {
		t.Errorf("Expected 2 services, received %d: %v", len(res.Services), res.Services)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Peers) != 2 {
		t.Fatalf("Expected 2 peers, received %d: %v", len(res.Peers), res.Peers)
	}

	if int(res.Peers[0].Direction) != int(ethpb.PeerDirection_INBOUND) {
		t.Errorf("Expected 1st peer to be an inbound (%d) connection, received %d", ethpb.PeerDirection_INBOUND, res.Peers[0].Direction)
	}
	if res.Peers[1].Direction != ethpb.PeerDirection_OUTBOUND {
		t.Errorf("Expected 2st peer to be an outbound (%d) connection, received %d", ethpb.PeerDirection_OUTBOUND, res.Peers[0].Direction)
	}
}
