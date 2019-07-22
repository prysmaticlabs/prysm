package rpc

import (
	"context"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/shared/version"
)

type mockSyncChecker struct {
	syncing bool
}

func (m *mockSyncChecker) Syncing() bool {
	return m.syncing
}

func TestNodeServer_GetSyncStatus(t *testing.T) {
	mSync := *mockSyncChecker{false}
	ns := &NodeServer{
		syncService: mSync,
	}
	res, err := ns.GetSyncStatus(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Syncing != mSync.syncing {
		t.Errorf("Wanted GetSyncStatus() = %d, received %d", mSync.syncing, res.Syncing)
	}
	mSync.syncing = true
	res, err = ns.GetSyncStatus(context.Background(), &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Syncing != mSync.syncing {
		t.Errorf("Wanted GetSyncStatus() = %d, received %d", mSync.syncing, res.Syncing)
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
		t.Errorf("Wanted GetVersion() = %d, received %d", v, res.Version)
	}
}
