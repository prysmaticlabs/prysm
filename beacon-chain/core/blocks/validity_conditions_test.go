package blocks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type mockDB struct {
	hasBlock bool
}

func (f *mockDB) HasBlock(ctx context.Context, h [32]byte) bool {
	return f.hasBlock
}

type mockPOWClient struct {
	blockExists bool
}

func (m *mockPOWClient) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	if m.blockExists {
		return &gethTypes.Block{}, nil
	}
	return nil, nil
}

func TestIsValidBlock_NoParent(t *testing.T) {
	beaconState := &pb.BeaconState{}

	ctx := context.Background()

	db := &mockDB{}
	powClient := &mockPOWClient{}

	beaconState.Slot = 3

	block := &ethpb.BeaconBlock{
		Slot: 4,
	}

	genesisTime := time.Unix(0, 0)

	db.hasBlock = false

	if err := IsValidBlock(ctx, beaconState, block,
		db.HasBlock, powClient.BlockByHash, genesisTime); err == nil {
		t.Fatal("block is valid despite not having a parent")
	}
}

func TestIsValidBlock_InvalidSlot(t *testing.T) {
	ctx := context.Background()
	beaconState := &pb.BeaconState{
		Slot: 3,
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: []byte{2},
			BlockHash:   []byte{3},
		},
	}
	db := &mockDB{
		hasBlock: true,
	}
	powClient := &mockPOWClient{
		blockExists: true,
	}
	block := &ethpb.BeaconBlock{
		Slot: 4,
	}
	genesisTime := time.Now()

	err := IsValidBlock(ctx, beaconState, block, db.HasBlock, powClient.BlockByHash, genesisTime)
	if err == nil {
		t.Fatalf("block is valid despite having an invalid slot %d", block.Slot)
	}
	if !strings.HasPrefix(err.Error(), "slot of block is too high: ") {
		t.Fatalf("expected the error about too high slot, but got an error: %v", err)
	}
}

func TestIsValidBlock_InvalidPoWReference(t *testing.T) {
	beaconState := &pb.BeaconState{}

	ctx := context.Background()

	db := &mockDB{}
	powClient := &mockPOWClient{}

	beaconState.Slot = 3

	block := &ethpb.BeaconBlock{
		Slot: 4,
	}

	genesisTime := time.Unix(0, 0)

	db.hasBlock = true
	block.Slot = 4
	powClient.blockExists = false
	beaconState.Eth1Data = &ethpb.Eth1Data{
		DepositRoot: []byte{2},
		BlockHash:   []byte{3},
	}

	if err := IsValidBlock(ctx, beaconState, block,
		db.HasBlock, powClient.BlockByHash, genesisTime); err == nil {
		t.Fatalf("block is valid despite having an invalid pow reference block")
	}

}
func TestIsValidBlock_InvalidGenesis(t *testing.T) {
	beaconState := &pb.BeaconState{}

	ctx := context.Background()

	db := &mockDB{}
	db.hasBlock = true

	powClient := &mockPOWClient{}
	powClient.blockExists = false

	beaconState.Slot = 3
	beaconState.Eth1Data = &ethpb.Eth1Data{
		DepositRoot: []byte{2},
		BlockHash:   []byte{3},
	}

	genesisTime := time.Unix(0, 0)
	block := &ethpb.BeaconBlock{
		Slot: 4,
	}

	invalidTime := time.Now().AddDate(1, 2, 3)

	if err := IsValidBlock(ctx, beaconState, block,
		db.HasBlock, powClient.BlockByHash, genesisTime); err == nil {
		t.Fatalf("block is valid despite having an invalid genesis time %v", invalidTime)
	}

}

func TestIsValidBlock_GoodBlock(t *testing.T) {
	beaconState := &pb.BeaconState{}
	ctx := context.Background()

	db := &mockDB{}
	db.hasBlock = true

	powClient := &mockPOWClient{}
	powClient.blockExists = true

	beaconState.Slot = 3
	beaconState.Eth1Data = &ethpb.Eth1Data{
		DepositRoot: []byte{2},
		BlockHash:   []byte{3},
	}

	genesisTime := time.Unix(0, 0)

	block := &ethpb.BeaconBlock{
		Slot: 4,
	}

	if err := IsValidBlock(ctx, beaconState, block,
		db.HasBlock, powClient.BlockByHash, genesisTime); err != nil {
		t.Fatal(err)
	}
}

func TestIsSlotValid(t *testing.T) {
	type testCaseStruct struct {
		slot        uint64
		genesisTime time.Time
		result      bool
	}

	testCases := []testCaseStruct{
		{
			slot:        5,
			genesisTime: roughtime.Now(),
			result:      false,
		},
		{
			slot: 5,
			genesisTime: roughtime.Now().Add(
				-time.Duration(params.BeaconConfig().SecondsPerSlot*5) * time.Second,
			),
			result: true,
		},
	}
	for _, testCase := range testCases {
		if testCase.result != IsSlotValid(testCase.slot, testCase.genesisTime) {
			t.Fatalf("invalid IsSlotValid result for %v", testCase)
		}
	}
}
