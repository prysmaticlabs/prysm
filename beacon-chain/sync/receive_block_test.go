package sync

import (
	"context"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestReceiveBlock_RecursivelyProcessesChildren(t *testing.T) {
	rs := NewRegularSyncService(context.Background(), DefaultRegularSyncConfig())

	parents := []*pb.BeaconBlock{
		{

		}
	}
    parentRoots := make([][]byte, len(parents))
    for i := range parents {
    	h, err := hashutil.HashBeaconBlock(parents[i])
    	if err != nil {
    		t.Fatal(err)
		}
		parentRoots[i] = h[:]
	}

	blocksMissingParent := []*pb.BeaconBlock{
		{
			Slot: params.BeaconConfig().GenesisSlot + 10,
			ParentRootHash32: parentRoots[0],
		},
		{
			Slot: params.BeaconConfig().GenesisSlot + 20,
			ParentRootHash32: parentRoots[1],
		},
	}
}
