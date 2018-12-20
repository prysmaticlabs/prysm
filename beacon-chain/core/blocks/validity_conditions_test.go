package blocks

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type mockDB struct {
	hasBlock       bool
	blockVoteCache utils.BlockVoteCache
}

func (f *mockDB) HasBlock(h [32]byte) bool {
	return f.hasBlock
}

func (f *mockDB) ReadBlockVoteCache(blockHashes [][32]byte) (utils.BlockVoteCache, error) {
	return f.blockVoteCache, nil
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

func TestBadBlock(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("failed to generate beacon state: %v", err)
	}

	ctx := context.Background()

	db := &mockDB{}
	powClient := &mockPOWClient{}

	beaconState.SetSlot(3)

	block := types.NewBlock(&pb.BeaconBlock{
		Slot: 4,
	})

	genesisTime := params.BeaconConfig().GenesisTime

	db.hasBlock = false

	if err := IsValidBlock(ctx, beaconState, block, true,
		db.HasBlock, powClient.BlockByHash, genesisTime); err == nil {
		t.Fatal("block is valid despite not having a parent")
	}

	block.Proto().Slot = 3
	db.hasBlock = true

	if err := IsValidBlock(ctx, beaconState, block, true,
		db.HasBlock, powClient.BlockByHash, genesisTime); err == nil {
		t.Fatalf("block is valid despite having an invalid slot %d", block.SlotNumber())
	}

	block.Proto().Slot = 4
	powClient.blockExists = false

	if err := IsValidBlock(ctx, beaconState, block, true,
		db.HasBlock, powClient.BlockByHash, genesisTime); err == nil {
		t.Fatalf("block is valid despite having an invalid pow reference block")
	}

	invalidTime := time.Now().AddDate(1, 2, 3)
	powClient.blockExists = false

	if err := IsValidBlock(ctx, beaconState, block, true,
		db.HasBlock, powClient.BlockByHash, genesisTime); err == nil {
		t.Fatalf("block is valid despite having an invalid genesis time %v", invalidTime)
	}

}

func TestValidBlock(t *testing.T) {
	beaconState, err := types.NewGenesisBeaconState(nil)
	if err != nil {
		t.Fatalf("failed to generate beacon state: %v", err)
	}

	ctx := context.Background()

	db := &mockDB{}
	powClient := &mockPOWClient{}

	beaconState.SetSlot(3)
	db.hasBlock = true

	block := types.NewBlock(&pb.BeaconBlock{
		Slot: 4,
	})

	genesisTime := params.BeaconConfig().GenesisTime
	powClient.blockExists = true

	if err := IsValidBlock(ctx, beaconState, block, true,
		db.HasBlock, powClient.BlockByHash, genesisTime); err != nil {
		t.Fatal(err)
	}

}
