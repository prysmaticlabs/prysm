package sync

import (
	"testing"
	"time"

	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mock"
	p2ptest "github.com/prysmaticlabs/prysm/shared/mock/p2p"
)

func TestService_StatusZeroEpoch(t *testing.T) {
	bState, err := stateTrie.InitializeFromProto(&pb.BeaconState{Slot: 0})
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p:         p2ptest.NewTestP2P(t),
		initialSync: new(mock.Sync),
		chain: &mock.ChainService{
			Genesis: time.Now(),
			State:   bState,
		},
	}
	r.chainStarted = true

	err = r.Status()
	if err != nil {
		t.Errorf("Wanted non failing status but got: %v", err)
	}
}
