package rpc

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/version"
	"google.golang.org/grpc"
)

var _ = serviceInfoFetcher(&grpc.Server{})

type mockSyncChecker struct {
	syncing bool
}

func (m *mockSyncChecker) Syncing() bool {
	return m.syncing
}

type mockInfoFetcher struct {
	serviceNames []string
}

func (m *mockInfoFetcher) GetServiceInfo() map[string]grpc.ServiceInfo {
	res := make(map[string]grpc.ServiceInfo)
	for _, s := range m.serviceNames {
		res[s] = grpc.ServiceInfo{}
	}
	return res
}

func TestNodeServer_GetSyncStatus(t *testing.T) {
	mSync := &mockSyncChecker{false}
	ns := &NodeServer{
		syncChecker: mSync,
	}
	res, err := ns.GetSyncStatus(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Syncing != mSync.syncing {
		t.Errorf("Wanted GetSyncStatus() = %v, received %v", mSync.syncing, res.Syncing)
	}
	mSync.syncing = true
	res, err = ns.GetSyncStatus(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Syncing != mSync.syncing {
		t.Errorf("Wanted GetSyncStatus() = %v, received %v", mSync.syncing, res.Syncing)
	}
}

func TestNodeServer_GetGenesis(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()
	addr := [20]byte{1, 2, 3, 4, 5, 6}
	beaconDB.VerifyContractAddress(ctx, common.Address(addr))
	beaconState := &pb.BeaconState{
		Slot:        0,
		GenesisTime: 0,
	}
	if err := beaconDB.SaveFinalizedState(beaconState); err != nil {
		t.Fatal(err)
	}

	ns := &NodeServer{
		beaconDB: beaconDB,
	}
	res, err := ns.GetGenesis(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(res.DepositContractAddress, addr[:]) {
		t.Errorf("Wanted GetGenesis().DepositContractAddress = %#x, received %#x", addr, res.DepositContractAddress)
	}
	genesisTimestamp := time.Unix(0, 0)
	protoTimestamp, err := ptypes.TimestampProto(genesisTimestamp)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(res.GenesisTime, protoTimestamp) {
		t.Errorf("Wanted GetGenesis().GenesisTime = %v, received %v", protoTimestamp, res.GenesisTime)
	}
}

func TestNodeServer_GetVersion(t *testing.T) {
	v := version.GetVersion()
	ns := &NodeServer{}
	res, err := ns.GetVersion(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Version != v {
		t.Errorf("Wanted GetVersion() = %s, received %s", v, res.Version)
	}
}

func TestNodeServer_GetImplementedServices(t *testing.T) {
	serviceNames := []string{"Validator", "Beacon Node", "Attestations"}
	mFetcher := &mockInfoFetcher{
		serviceNames,
	}
	ns := &NodeServer{
		serviceFetcher: mFetcher,
	}
	res, err := ns.ListImplementedServices(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Services) == 0 {
		t.Errorf("Expected %d services, received %d", len(serviceNames), len(res.Services))
	}
	if reflect.DeepEqual(res.Services, serviceNames) {
		t.Error("Expected list of services to be sorted")
	}
	sort.Strings(serviceNames)
	if !reflect.DeepEqual(res.Services, serviceNames) {
		t.Errorf("Wanted ListImplementedServices() = %v, received %v", serviceNames, res.Services)
	}
}
