package helpers_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func setupInitialDeposits(numDeposits int) []*pb.Deposit {
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		balance := params.BeaconConfig().MaxDepositAmount
		depositData := &pb.DepositData{
			Pubkey: []byte(strconv.Itoa(i)),
			Amount: balance,
		}

		deposits[i] = &pb.Deposit{
			Data:  depositData,
			Index: uint64(i),
		}
	}
	return deposits
}

func TestAttestationDataSlot_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	deposits := setupInitialDeposits(100)
	if err := db.InitializeState(context.Background(), uint64(0), deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	offset := uint64(0)
	committeeCount := helpers.EpochCommitteeCount(beaconState, 0)
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
