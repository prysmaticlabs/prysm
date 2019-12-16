package db

import (
	"math/big"
	// "reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	// ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func TestSetProposedForEpoch_SetsBit(t *testing.T) {
	c := params.BeaconConfig()
	c.WeakSubjectivityPeriod = 128
	params.OverrideBeaconConfig(c)

	proposals := &ValidatorProposalHistory{
		ProposalHistory: big.NewInt(0),
		EpochAtFirstBit: 0,
	}
	proposals.SetProposedForEpoch(4)
	proposed := proposals.HasProposedForEpoch(4)
	if !proposed {
		t.Fatal("fuck")
	}
	// Make sure no other bits are changed.
	for i := uint64(0); i < c.WeakSubjectivityPeriod; i++ {
		if i == 4 {
			continue
		}
		proposed = proposals.HasProposedForEpoch(i)
		if proposed {
			t.Fatal("fuck")
		}
	}
}

func TestSetProposedForEpoch_PrunesOverWSPeriod(t *testing.T) {
	c := params.BeaconConfig()
	c.WeakSubjectivityPeriod = 128
	params.OverrideBeaconConfig(c)

	epoch := uint64(132)
	proposals := &ValidatorProposalHistory{
		ProposalHistory: big.NewInt(0),
		EpochAtFirstBit: 0,
	}
	proposals.SetProposedForEpoch(epoch)

	if proposals.EpochAtFirstBit != 4 {
		t.Fatal(proposals.EpochAtFirstBit)
	}

	proposed := proposals.HasProposedForEpoch(epoch)
	if !proposed {
		t.Fatal("fuck")
	}
	// Make sure no other bits are changed.
	for i := uint64(proposals.EpochAtFirstBit); i < c.WeakSubjectivityPeriod+proposals.EpochAtFirstBit; i++ {
		if i == epoch {
			continue
		}
		proposed = proposals.HasProposedForEpoch(i)
		if proposed {
			t.Fatal("fuck")
		}
	}
}

func TestSetProposedForEpoch_54KEpochsPrunes(t *testing.T) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	proposals := &ValidatorProposalHistory{
		ProposalHistory: big.NewInt(0),
		EpochAtFirstBit: 0,
	}
	randomIndexes := []uint64{23, 423, 8900, 11347, 25033, 52225, 53999}
	for i := 0; i < len(randomIndexes); i++ {
		proposals.SetProposedForEpoch(randomIndexes[i])
	}
	if proposals.EpochAtFirstBit != 0 {
		t.Fatal(proposals.EpochAtFirstBit)
	}

	// Make sure no other bits are changed.
	for i := uint64(0); i < params.BeaconConfig().WeakSubjectivityPeriod; i++ {
		setIndex := false
		for r := 0; r < len(randomIndexes); r++ {
			if i == randomIndexes[r] {
				setIndex = true
			}
		}
		if !setIndex {
			proposed := proposals.HasProposedForEpoch(i)
			if proposed {
				t.Fatal("fuck")
			}
		} else {
			proposed := proposals.HasProposedForEpoch(i)
			if !proposed {
				t.Fatal("fuck")
			}
		}
	}

	// Set proposed after 54K epochs to prune.
	proposals.SetProposedForEpoch(wsPeriod + randomIndexes[1])
	if proposals.EpochAtFirstBit != randomIndexes[1] {
		t.Fatal("fuck")
	}
	// Should be able to access epoch at first bit.
	if !proposals.HasProposedForEpoch(randomIndexes[1]) {
		t.Fatal("fuck")
	}
}

// func TestNilDBHistoryBlkHdr(t *testing.T) {
// 	db := SetupDB(t)
// 	defer TeardownDB(t, db)

// 	epoch := uint64(1)
// 	validatorID := uint64(1)

// 	hasBlockHeader := db.HasBlockHeader(epoch, validatorID)
// 	if hasBlockHeader {
// 		t.Fatal("HasBlockHeader should return false")
// 	}

// 	bPrime, err := db.BlockHeader(epoch, validatorID)
// 	if err != nil {
// 		t.Fatalf("failed to get block: %v", err)
// 	}
// 	if bPrime != nil {
// 		t.Fatalf("get should return nil for a non existent key")
// 	}
// }

// func TestSaveHistoryBlkHdr(t *testing.T) {
// 	db := SetupDB(t)
// 	defer TeardownDB(t, db)
// 	tests := []struct {
// 		epoch uint64
// 		vID   uint64
// 		bh    *ethpb.BeaconBlockHeader
// 	}{
// 		{
// 			epoch: uint64(0),
// 			vID:   uint64(0),
// 			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in")},
// 		},
// 		{
// 			epoch: uint64(0),
// 			vID:   uint64(1),
// 			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 2nd")},
// 		},
// 		{
// 			epoch: uint64(1),
// 			vID:   uint64(0),
// 			bh:    &ethpb.BeaconBlockHeader{Signature: []byte("let me in 3rd")},
// 		},
// 	}

// 	for _, tt := range tests {
// 		err := db.SaveBlockHeader(tt.epoch, tt.vID, tt.bh)
// 		if err != nil {
// 			t.Fatalf("save block failed: %v", err)
// 		}

// 		bha, err := db.BlockHeader(tt.epoch, tt.vID)
// 		if err != nil {
// 			t.Fatalf("failed to get block: %v", err)
// 		}

// 		if bha == nil || !reflect.DeepEqual(bha[0], tt.bh) {
// 			t.Fatalf("get should return bh: %v", bha)
// 		}
// 	}
//}

// func TestDeleteHistoryBlkHdr(t *testing.T) {

// }

// func TestHasHistoryBlkHdr(t *testing.T) {
// }
