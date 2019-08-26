package rpc

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type mockSyncChecker struct {
	syncing bool
}

func (m *mockSyncChecker) Syncing() bool {
	return m.syncing
}

func (m *mockSyncChecker) Status() error {
	return nil
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
	server := grpc.NewServer()
	ns := &NodeServer{
		server: server,
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
