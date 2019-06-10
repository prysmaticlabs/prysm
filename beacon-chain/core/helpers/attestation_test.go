package helpers_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestAttestationDataSlot_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	if err := db.InitializeState(context.Background(), uint64(0), deposits, nil); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	offset := uint64(0)
	committeeCount, _ := helpers.EpochCommitteeCount(beaconState, 0)
	expect := offset / (committeeCount / params.BeaconConfig().SlotsPerEpoch)
	attSlot, err := helpers.AttestationDataSlot(beaconState, &pb.AttestationData{
		TargetEpoch: 0,
		Crosslink: &pb.Crosslink{
			Shard: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if attSlot != expect {
		t.Errorf("Expected %d, received %d", expect, attSlot)
	}
}
