package blockchain

import (
	"context"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestApplyForkChoice_ChainSplitReorg(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	beaconState := &pb.BeaconState{
		Slot: 10,
		ValidatorBalances: []uint64{
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount},
		ValidatorRegistry: []*pb.Validator{{}, {}, {}, {}},
	}

	chainService := setupBeaconChain(t, beaconDB, nil)

	// Construct a forked chain that looks as follows:
	//    /- B1 - B3 - B5 (current head)
	// B0  - B2 - B4
	blocks, roots := constructForkedChain(t, chainService, beaconState)

	// Give block 4 the most votes (2).
	voteTargets := make(map[uint64]*pb.AttestationTarget)
	voteTargets[0] = &pb.AttestationTarget{
		Slot:       blocks[5].Slot,
		BlockRoot:  roots[5][:],
		ParentRoot: blocks[5].ParentRootHash32,
	}
	voteTargets[1] = &pb.AttestationTarget{
		Slot:       blocks[4].Slot,
		BlockRoot:  roots[4][:],
		ParentRoot: blocks[4].ParentRootHash32,
	}
	voteTargets[2] = &pb.AttestationTarget{
		Slot:       blocks[4].Slot,
		BlockRoot:  roots[4][:],
		ParentRoot: blocks[4].ParentRootHash32,
	}
	// LMDGhost should pick block 5.
	head, err := chainService.lmdGhost(context.Background(), blocks[0], beaconState, voteTargets)
	if err != nil {
		t.Fatalf("Could not run LMD GHOST: %v", err)
	}
	if !reflect.DeepEqual(blocks[4], head) {
		t.Errorf("Expected head to equal %v, received %v", blocks[4], head)
	}
}

func constructForkedChain(t *testing.T, chainService *ChainService, beaconState *pb.BeaconState) ([]*pb.BeaconBlock, [][32]byte) {
	// Construct the following chain:
	//    /- B1 - B3 - B5
	// B0  - B2 - B4 (State is slot 10)
	ctx := context.Background()
	blocks := make([]*pb.BeaconBlock, 6)
	roots := make([][32]byte, 6)
	var err error
	blocks[0] = &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: []byte{'A'},
	}
	roots[0], err = hashutil.HashBeaconBlock(blocks[0])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(blocks[0]); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(ctx, blocks[0], beaconState); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	blocks[1] = &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: roots[0][:],
	}
	roots[1], err = hashutil.HashBeaconBlock(blocks[1])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(blocks[1]); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(ctx, blocks[1], beaconState); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	blocks[2] = &pb.BeaconBlock{
		Slot:             3,
		ParentRootHash32: roots[0][:],
	}
	roots[2], err = hashutil.HashBeaconBlock(blocks[2])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(blocks[2]); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(ctx, blocks[2], beaconState); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	blocks[3] = &pb.BeaconBlock{
		Slot:             4,
		ParentRootHash32: roots[1][:],
	}
	roots[3], err = hashutil.HashBeaconBlock(blocks[3])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(blocks[3]); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(ctx, blocks[3], beaconState); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	blocks[4] = &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: roots[2][:],
	}
	roots[4], err = hashutil.HashBeaconBlock(blocks[4])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(blocks[4]); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(ctx, blocks[4], beaconState); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}

	blocks[5] = &pb.BeaconBlock{
		Slot:             6,
		ParentRootHash32: roots[3][:],
	}
	roots[5], err = hashutil.HashBeaconBlock(blocks[5])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = chainService.beaconDB.SaveBlock(blocks[5]); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	if err = chainService.beaconDB.UpdateChainHead(ctx, blocks[5], beaconState); err != nil {
		t.Fatalf("Could update chain head: %v", err)
	}
	return blocks, roots
}
