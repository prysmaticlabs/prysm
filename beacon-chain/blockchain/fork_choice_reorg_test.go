package blockchain

import (
	"context"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockAttestationHandler struct {
	targets map[uint64]*pb.AttestationTarget
}

func (m *mockAttestationHandler) LatestAttestationTarget(beaconState *pb.BeaconState, idx uint64) (*pb.AttestationTarget, error) {
	return m.targets[idx], nil
}

func (m *mockAttestationHandler) BatchUpdateLatestAttestation(ctx context.Context, atts []*pb.Attestation) error {
	return nil
}

func TestApplyForkChoice_ChainSplitReorg(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	ctx := context.Background()
	deposits, _ := setupInitialDeposits(t, 100)
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	justifiedState, err := state.GenesisBeaconState(deposits, 0, eth1Data)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}

	chainService := setupBeaconChain(t, beaconDB, nil)

	// Construct a forked chain that looks as follows:
	//    /------B1 ----B3 ----- B5 (current head)
	// B0 --B2 -------------B4
	blocks, roots := constructForkedChain(t, justifiedState)

	// We then setup a canonical chain of the following blocks:
	// B0->B1->B3->B5.
	if err := chainService.beaconDB.SaveBlock(blocks[0]); err != nil {
		t.Fatal(err)
	}
	justifiedState.LatestBlock = blocks[0]
	if err := chainService.beaconDB.SaveJustifiedState(justifiedState); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.SaveJustifiedBlock(blocks[0]); err != nil {
		t.Fatal(err)
	}
	if err := chainService.beaconDB.UpdateChainHead(ctx, blocks[0], justifiedState); err != nil {
		t.Fatal(err)
	}
	canonicalBlockIndices := []int{1, 3, 5}
	postState := proto.Clone(justifiedState).(*pb.BeaconState)
	for _, canonicalIndex := range canonicalBlockIndices {
		postState, err = chainService.ApplyBlockStateTransition(ctx, blocks[canonicalIndex], postState)
		if err != nil {
			t.Fatal(err)
		}
		if err := chainService.beaconDB.SaveBlock(blocks[canonicalIndex]); err != nil {
			t.Fatal(err)
		}
		if err := chainService.beaconDB.UpdateChainHead(ctx, blocks[canonicalIndex], postState); err != nil {
			t.Fatal(err)
		}
	}

	chainHead, err := chainService.beaconDB.ChainHead()
	if err != nil {
		t.Fatal(err)
	}
	if chainHead.Slot != justifiedState.Slot+5 {
		t.Errorf(
			"Expected chain head with slot %d, received %d",
			justifiedState.Slot+5-params.BeaconConfig().GenesisSlot,
			chainHead.Slot-params.BeaconConfig().GenesisSlot,
		)
	}

	// We then save forked blocks and their historical states (but do not update chain head).
	// The fork is from B0->B2->B4.
	forkedBlockIndices := []int{2, 4}
	forkState := proto.Clone(justifiedState).(*pb.BeaconState)
	for _, forkIndex := range forkedBlockIndices {
		forkState, err = chainService.ApplyBlockStateTransition(ctx, blocks[forkIndex], forkState)
		if err != nil {
			t.Fatal(err)
		}
		if err := chainService.beaconDB.SaveBlock(blocks[forkIndex]); err != nil {
			t.Fatal(err)
		}
		if err := chainService.beaconDB.SaveHistoricalState(ctx, forkState); err != nil {
			t.Fatal(err)
		}
	}

	// Give the block from the forked chain, B4, the most votes.
	voteTargets := make(map[uint64]*pb.AttestationTarget)
	voteTargets[0] = &pb.AttestationTarget{
		Slot:       blocks[5].Slot,
		BlockRoot:  roots[5][:],
		ParentRoot: blocks[5].ParentRootHash32,
	}
	for i := 1; i < len(deposits); i++ {
		voteTargets[uint64(i)] = &pb.AttestationTarget{
			Slot:       blocks[4].Slot,
			BlockRoot:  roots[4][:],
			ParentRoot: blocks[4].ParentRootHash32,
		}
	}
	attHandler := &mockAttestationHandler{
		targets: voteTargets,
	}
	chainService.attsService = attHandler

	block4State, err := chainService.beaconDB.HistoricalStateFromSlot(ctx, blocks[4].Slot)
	if err != nil {
		t.Fatal(err)
	}
	// Applying the fork choice rule should reorg to B4 successfully.
	if err := chainService.ApplyForkChoiceRule(ctx, blocks[4], block4State); err != nil {
		t.Fatal(err)
	}

	newHead, err := chainService.beaconDB.ChainHead()
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(newHead, blocks[4]) {
		t.Errorf(
			"Expected chain head %v, received %v",
			blocks[4],
			newHead,
		)
	}
	want := fmt.Sprintf(
		"Reorg happened, last head at slot %d, new head block at slot %d",
		blocks[5].Slot-params.BeaconConfig().GenesisSlot, blocks[4].Slot-params.BeaconConfig().GenesisSlot,
	)
	testutil.AssertLogsContain(t, hook, want)
}

func constructForkedChain(t *testing.T, beaconState *pb.BeaconState) ([]*pb.BeaconBlock, [][32]byte) {
	// Construct the following chain:
	//    /------B1 ----B3 ----- B5 (current head)
	// B0 --B2 -------------B4
	blocks := make([]*pb.BeaconBlock, 6)
	roots := make([][32]byte, 6)
	var err error
	blocks[0] = &pb.BeaconBlock{
		Slot:             beaconState.Slot,
		ParentRootHash32: []byte{'A'},
		Body:             &pb.BeaconBlockBody{},
	}
	roots[0], err = hashutil.HashBeaconBlock(blocks[0])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}

	blocks[1] = &pb.BeaconBlock{
		Slot:             beaconState.Slot + 2,
		ParentRootHash32: roots[0][:],
		Body:             &pb.BeaconBlockBody{},
	}
	roots[1], err = hashutil.HashBeaconBlock(blocks[1])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}

	blocks[2] = &pb.BeaconBlock{
		Slot:             beaconState.Slot + 1,
		ParentRootHash32: roots[0][:],
		Body:             &pb.BeaconBlockBody{},
	}
	roots[2], err = hashutil.HashBeaconBlock(blocks[2])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}

	blocks[3] = &pb.BeaconBlock{
		Slot:             beaconState.Slot + 3,
		ParentRootHash32: roots[1][:],
		Body:             &pb.BeaconBlockBody{},
	}
	roots[3], err = hashutil.HashBeaconBlock(blocks[3])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}

	blocks[4] = &pb.BeaconBlock{
		Slot:             beaconState.Slot + 4,
		ParentRootHash32: roots[2][:],
		Body:             &pb.BeaconBlockBody{},
	}
	roots[4], err = hashutil.HashBeaconBlock(blocks[4])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}

	blocks[5] = &pb.BeaconBlock{
		Slot:             beaconState.Slot + 5,
		ParentRootHash32: roots[3][:],
		Body:             &pb.BeaconBlockBody{},
	}
	roots[5], err = hashutil.HashBeaconBlock(blocks[5])
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	return blocks, roots
}
