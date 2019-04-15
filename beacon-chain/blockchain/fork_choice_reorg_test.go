package blockchain

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/params"
	"testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"time"
)

func generateBlocksWithAttestations(t *testing.T, ctx context.Context, beaconDB *db.BeaconDB) []*pb.BeaconBlock {
	beaconState, err := beaconDB.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("State: %v", beaconState)
	return nil
}

// This function tests the following: when two nodes A and B are running at slot 10
// and node A reorgs back to slot 7 (an epoch boundary), while node B remains the same,
// once the nodes catch up a few blocks later, we expect their state and validator
// balances to remain the same. That is, we expect no deviation in validator balances.
func TestEpochReorg_MatchingStates(t *testing.T) {
	// First we setup two independent db's for node A and B.
	ctx := context.Background()
	beaconDB1 := internal.SetupDB(t)
	beaconDB2 := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB1)
	defer internal.TeardownDB(t, beaconDB2)

	//chainService1 := setupBeaconChain(t, beaconDB1, nil)
	//chainService2 := setupBeaconChain(t, beaconDB2, nil)
	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 8)
	if err := beaconDB1.InitializeState(ctx, unixTime, deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	if err := beaconDB2.InitializeState(ctx, unixTime, deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	genesisBlock, err := beaconDB1.BlockBySlot(ctx, params.BeaconConfig().GenesisSlot)
	if err != nil {
		t.Fatal(err)
	}

	// Then, we create the chain up to slot 10 in both.
    blocks := []*pb.BeaconBlock{}
	for i := 1; i <= 10; i++ {
		block := &pb.BeaconBlock{
			Slot:             beaconState.Slot + 1,
			RandaoReveal:     epochSignature.Marshal(),
			ParentRootHash32: prevBlockRoot[:],
			StateRootHash32:  []byte{},
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{},
				BlockHash32:       []byte{},
			},
			Body: &pb.BeaconBlockBody{
				Attestations:      []*pb.Attestation{},
			},
		}
	}

	// We update attestation targets for node A such that validators point to the block
	// at slot 7 as canonical - then, a reorg to that slot will occur.

	// We then proceed in both nodes normally through several blocks.

	// At this point, once the two nodes are fully caught up, we expect their state,
	// in particular their balances, to be equal.
}
