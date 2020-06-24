package node

import (
	"bytes"
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
	db, _ := dbutil.SetupDB(t)
	ctx := context.Background()
	addr := common.Address{1, 2, 3}
	if err := db.SaveDepositContractAddress(ctx, addr); err != nil {
		t.Fatal(err)
	}
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
	if !bytes.Equal(genValRoot[:], res.GenesisValidatorsRoot) {
		t.Errorf("Wanted GenesisValidatorsRoot = %v, received %v", genValRoot, res.GenesisValidatorsRoot)
	}

	ns.GenesisTimeFetcher = &mock.ChainService{Genesis: time.Unix(10, 0)}
	res, err = ns.GetGenesis(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	pUnix, err = ptypes.TimestampProto(time.Unix(10, 0))
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

func TestNodeServer_GetHost(t *testing.T) {
	server := grpc.NewServer()
	peersProvider := &mockP2p.MockPeersProvider{}
	mP2P := mockP2p.NewTestP2P(t)
	key, err := crypto.GenerateKey()
	db, err := enode.OpenDB("")
	if err != nil {
		t.Fatal("could not open node's peer database")
	}
	lNode := enode.NewLocalNode(db, key)
	record := lNode.Node().Record()
	stringENR, err := p2p.SerializeENR(record)
	if err != nil {
		t.Fatal(err)
	}
	ns := &Server{
		PeerManager:  &mockP2p.MockPeerManager{BHost: mP2P.BHost, Enr: record, PID: mP2P.BHost.ID()},
		PeersFetcher: peersProvider,
	}
	ethpb.RegisterNodeServer(server, ns)
	reflection.Register(server)
	h, err := ns.GetHost(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if h.PeerId != mP2P.PeerID().String() {
		t.Errorf("Wanted Peer id of %s but got %s", mP2P.PeerID().String(), h.PeerId)
	}
	if h.Enr != stringENR {
		t.Errorf("Wanted %s for enr but couldn't get it %s", stringENR, h.Enr)
	}
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
