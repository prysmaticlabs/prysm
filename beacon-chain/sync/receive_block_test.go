package sync

import (
	"context"
	"github.com/prysmaticlabs/prysm/shared/params"
	"testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestReceiveBlock_RecursivelyProcessesChildren(t *testing.T) {
	rs := NewRegularSyncService(context.Background(), DefaultRegularSyncConfig())

	parents := []*pb.BeaconBlock{

	}

	blocksMissingParent := []*pb.BeaconBlock{
		{
			Slot: params.BeaconConfig().GenesisSlot + 10,
			ParentRootHash32: []byte("dad1"),
		},
		{
			Slot: params.BeaconConfig().GenesisSlot + 20,
			ParentRootHash32: []byte("dad2"),
		},
	}
}
