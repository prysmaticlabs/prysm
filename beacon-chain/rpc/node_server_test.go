package rpc

import (
	"context"
	"reflect"
	"sort"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
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
