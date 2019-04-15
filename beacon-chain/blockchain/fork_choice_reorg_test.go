package blockchain

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"testing"
	"time"
)

type mockAttestationManager struct {
	targets map[uint64]*pb.BeaconBlock
	beaconDB *db.BeaconDB
}

func (m *mockAttestationManager) LatestAttestationTarget(ctx context.Context, validatorIndex uint64) (*pb.BeaconBlock, error) {
	return m.targets[validatorIndex], nil
}
func (m *mockAttestationManager) UpdateLatestAttestation(ctx context.Context, attestation *pb.Attestation) error {
	state, err := m.beaconDB.HeadState(ctx)
	if err != nil {
		return err
	}
	var committee []uint64
	// We find the crosslink committee for the shard and slot by the attestation.
	committees, err := helpers.CrosslinkCommitteesAtSlot(state, attestation.Data.Slot, false /* registryChange */)
	if err != nil {
		return err
	}

	// Find committee for shard.
	for _, v := range committees {
		if v.Shard == attestation.Data.Shard {
			committee = v.Committee
			break
		}
	}
	// The participation bitfield from attestation is represented in bytes,
	// here we multiply by 8 to get an accurate validator count in bits.
	bitfield := attestation.AggregationBitfield
	totalBits := len(bitfield) * 8

	// Check each bit of participation bitfield to find out which
	// attester has submitted new attestation.
	// This is has O(n) run time and could be optimized down the line.
	for i := 0; i < totalBits; i++ {
		bitSet, err := bitutil.CheckBit(bitfield, i)
		if err != nil {
			return err
		}
		if !bitSet {
			continue
		}

		root := bytesutil.ToBytes32(attestation.Data.BeaconBlockRootHash32)
		targetBlock, err := m.beaconDB.Block(root)
		if err != nil {
			return fmt.Errorf("could not get target block: %v", err)
		}
		m.targets[committee[i]] = targetBlock
		return nil
	}
	return nil
}

type assignment struct {
	shard uint64
	validatorIndex uint64
	committee []uint64
}

func createBlock(
	t *testing.T, slot uint64, blocks []*pb.BeaconBlock, states []*pb.BeaconState, assignments map[uint64]*assignment,
) *pb.BeaconBlock {
	if slot % params.BeaconConfig().SlotsPerEpoch == 0 {
		for idx := range states[slot-1].ValidatorRegistry {
			committee, shard, slot, _, err :=
				helpers.CommitteeAssignment(states[slot-1], params.BeaconConfig().GenesisSlot+slot, uint64(idx), false)
			if err != nil {
				t.Fatal(err)
			}
			assignments[slot] = &assignment{
				shard,
				uint64(idx),
				committee,
			}
		}
	}
	parent := blocks[slot-1]
	prevBlockRoot, err := hashutil.HashBeaconBlock(parent)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Parent slot: %d, parent root %#x\n", slot-1, prevBlockRoot)
	block := &pb.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + slot,
		RandaoReveal:     []byte{},
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

	// We generate attestation using the previous slot due to the MIN_ATTESTATION_INCLUSION_DELAY.
	prevSlot := params.BeaconConfig().GenesisSlot + slot-1
	committee := assignments[prevSlot].committee
	shard := assignments[prevSlot].shard
	attestation := &pb.Attestation{
		Data: &pb.AttestationData{},
	}
	attestation.CustodyBitfield = make([]byte, len(committee))
	// Find the index in committee to be used for the aggregation bitfield.
	var indexInCommittee int
	for j, vIndex := range committee {
		if vIndex == assignments[prevSlot].validatorIndex {
			indexInCommittee = j
			break
		}
	}
	aggregationBitfield := bitutil.SetBitfield(indexInCommittee, len(committee))
	attestation.AggregationBitfield = aggregationBitfield
	attestation.AggregateSignature = []byte("signed")

	epochBoundaryRoot := make([]byte, 32)
	epochStartSlot := helpers.StartSlot(helpers.SlotToEpoch(prevSlot))
	if epochStartSlot == prevSlot {
		epochBoundaryRoot = prevBlockRoot[:]
	} else {
		epochBoundaryRoot, err = b.BlockRoot(states[slot-1], epochStartSlot)
		if err != nil {
			t.Fatal(err)
		}
	}
	// epoch_start_slot = get_epoch_start_slot(slot_to_epoch(head.slot))
	// Fetch the justified block root = hash_tree_root(justified_block) where
	// justified_block is the block at state.justified_epoch in the chain defined by head.
	// On the server side, this is fetched by calling get_block_root(state, justified_epoch).
	// If the last justified boundary slot is the same as state current slot (ex: slot 0),
	// we set justified block root to an empty root.
	justifiedBlockRoot := states[slot-1].JustifiedRoot

	// If an attester has to attest for genesis block.
	if states[slot-1].Slot == params.BeaconConfig().GenesisSlot {
		epochBoundaryRoot = params.BeaconConfig().ZeroHash[:]
		justifiedBlockRoot = params.BeaconConfig().ZeroHash[:]
	}
	attestation.Data.Slot = prevSlot
	attestation.Data.Shard = shard
	attestation.Data.EpochBoundaryRootHash32 = epochBoundaryRoot
	attestation.Data.JustifiedBlockRootHash32 = justifiedBlockRoot
	attestation.Data.JustifiedEpoch = states[slot-1].JustifiedEpoch
	attestation.Data.LatestCrosslink = states[slot-1].LatestCrosslinks[shard]
	attestation.Data.CrosslinkDataRootHash32 = params.BeaconConfig().ZeroHash[:]
	block.Body.Attestations = []*pb.Attestation{attestation}
	return block
}

func advanceChain(
	t *testing.T,
	chainService *ChainService,
	genesisBlock *pb.BeaconBlock,
	genesisState *pb.BeaconState,
	assignments map[uint64]*assignment,
	numBlocks int,
) ([]*pb.BeaconBlock, []*pb.BeaconState) {
	// Then, we create the chain up to slot 100 in both.
	blocks := make([]*pb.BeaconBlock, numBlocks+1)
	blocks[0] = genesisBlock
	states := make([]*pb.BeaconState, numBlocks+1)
	states[0] = genesisState
	for idx := range genesisState.ValidatorRegistry {
		committee, shard, slot, _, err :=
			helpers.CommitteeAssignment(genesisState, genesisBlock.Slot, uint64(idx), false)
		if err != nil {
			t.Fatal(err)
		}
		assignments[slot] = &assignment{
			shard,
			uint64(idx),
			committee,
		}
	}
	for i := uint64(1); i <= uint64(numBlocks); i++ {
		block := createBlock(t, i, blocks, states, assignments)
		beaconState, err := chainService.ApplyBlockStateTransition(context.Background(), block, states[i-1])
		if err != nil {
			t.Fatal(err)
		}
		if err := chainService.beaconDB.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		if err := chainService.beaconDB.UpdateChainHead(context.Background(), block, beaconState); err != nil {
			t.Fatal(err)
		}
		blocks[i] = block
		states[i] = beaconState
	}
	return blocks, states
}

// This function tests the following: when two nodes A and B are running at slot 10
// and node A reorgs back to slot 7 (an epoch boundary), while node B remains the same,
// once the nodes catch up a few blocks later, we expect their state and validator
// balances to remain the same. That is, we expect no deviation in validator balances.
func TestEpochReorg_MatchingStates(t *testing.T) {
	params.UseDemoBeaconConfig()
	ctx := context.Background()
	// First we setup two independent db's for node A and B.
	beaconDB1 := internal.SetupDB(t)
	beaconDB2 := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB1)
	defer internal.TeardownDB(t, beaconDB2)

	attService1 := &mockAttestationManager{
		beaconDB: beaconDB1,
		targets: make(map[uint64]*pb.BeaconBlock),
	}
	attService2 := &mockAttestationManager{
		beaconDB: beaconDB2,
		targets: make(map[uint64]*pb.BeaconBlock),
	}
	chainService1 := setupBeaconChain(t, beaconDB1, attService1)
	chainService2 := setupBeaconChain(t, beaconDB2, attService2)
	assignments1 := make(map[uint64]*assignment)
	assignments2 := make(map[uint64]*assignment)

	unixTime := time.Unix(0, 0)
	deposits, _ := setupInitialDeposits(t, 8)
	genesisState1, err := chainService1.initializeBeaconChain(unixTime, deposits, &pb.Eth1Data{})
	if err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	genesisState2, err := chainService2.initializeBeaconChain(unixTime, deposits, &pb.Eth1Data{})
	if err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	stateRoot1, err := hashutil.HashProto(genesisState1)
	if err != nil {
		t.Fatal(err)
	}
	stateRoot2, err := hashutil.HashProto(genesisState2)
	if err != nil {
		t.Fatal(err)
	}
	genesisBlock1 := b.NewGenesisBlock(stateRoot1[:])
	genesisBlock2 := b.NewGenesisBlock(stateRoot2[:])

	if !proto.Equal(genesisBlock1, genesisBlock2) {
		t.Errorf("Expected genesis blocks to match, got %v = %v", genesisBlock1, genesisBlock2)
	}

	// We then advance the chain.
	blocks1, states1 := advanceChain(t, chainService1, genesisBlock1, genesisState1,assignments1, 34)
	blocks2, states2 := advanceChain(t, chainService2, genesisBlock2, genesisState2, assignments2, 34)

	lastState1 := states1[len(states1)-1]
	lastState2 := states2[len(states2)-1]

	// We expect both nodes to have matching balances after generating N
	// blocks, including attestations, and applying N state transitions.
	balancesNodeA := lastState1.ValidatorBalances
	balancesNodeB := lastState2.ValidatorBalances
	for i := range balancesNodeA {
		if balancesNodeA[i] != balancesNodeB[i] {
			t.Errorf(
				"Expected balance to match at index %d, received %v = %v",
				i,
				balancesNodeA[i],
				balancesNodeB[i],
			)
		}
	}
	t.Logf("Validator balances node A: %v", balancesNodeA)
	t.Logf("Finalized epoch node A: %v", lastState1.FinalizedEpoch-params.BeaconConfig().GenesisEpoch)
	t.Logf("Justified epoch node A: %v", lastState1.JustifiedEpoch-params.BeaconConfig().GenesisEpoch)
	t.Logf("Validator balances node B: %v", balancesNodeB)
	t.Logf("Finalized epoch node B: %v", lastState2.FinalizedEpoch-params.BeaconConfig().GenesisEpoch)
	t.Logf("Justified epoch node B: %v", lastState2.JustifiedEpoch-params.BeaconConfig().GenesisEpoch)

	// We update attestation targets for node A such that validators point to the block
	// at slot 31 as canonical - then, a reorg to that slot will occur.
	for idx := range lastState1.ValidatorRegistry {
        attService1.targets[uint64(idx)] = blocks1[30]
	}

	newBlock1 := createBlock(t, 35, blocks1, states1, assignments1)
	newBlock2 := createBlock(t, 35, blocks2, states2, assignments2)
	blocks1 = append(blocks1, newBlock1)
	blocks2 = append(blocks2, newBlock2)

	// We force node A to go through a reorg.
	if _, err := chainService1.ApplyBlockStateTransition(ctx, newBlock1, states1[34]); err != nil {
		t.Fatal(err)
	}
	if err := chainService1.beaconDB.SaveBlock(newBlock1); err != nil {
		t.Fatal(err)
	}
	oldState, err := chainService1.beaconDB.HistoricalStateFromSlot(ctx, blocks1[30].Slot)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService1.beaconDB.UpdateChainHead(ctx, blocks1[30], oldState); err != nil {
		t.Fatal(err)
	}

	// Node B does not go through a reorg.
	beaconState2, err := chainService2.ApplyBlockStateTransition(ctx, newBlock2, states2[34])
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService2.beaconDB.SaveBlock(newBlock2); err != nil {
		t.Fatal(err)
	}
	if err := chainService2.beaconDB.UpdateChainHead(ctx, newBlock2, beaconState2); err != nil {
		t.Fatal(err)
	}
	states2 = append(states2, beaconState2)

	// We look at chain heads.
	nodeAHead, err := chainService1.beaconDB.ChainHead()
	if err != nil {
		t.Fatal(err)
	}
	nodeBHead, err := chainService2.beaconDB.ChainHead()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Node A head slot: %v", nodeAHead.Slot-params.BeaconConfig().GenesisSlot)
	t.Logf("Node B head slot: %v", nodeBHead.Slot-params.BeaconConfig().GenesisSlot)

	// Now we generate block with slot 36 and force both nodes to go through the block transition.
	newBlock := createBlock(t, 36, blocks2, states2, assignments2)
	beaconState1, err := chainService1.ApplyBlockStateTransition(ctx, newBlock, oldState)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService1.beaconDB.SaveBlock(newBlock); err != nil {
		t.Fatal(err)
	}
	if err := chainService1.beaconDB.UpdateChainHead(ctx, newBlock, beaconState1); err != nil {
		t.Fatal(err)
	}
	beaconState2, err = chainService2.ApplyBlockStateTransition(ctx, newBlock, states2[35])
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService2.beaconDB.SaveBlock(newBlock); err != nil {
		t.Fatal(err)
	}
	if err := chainService2.beaconDB.UpdateChainHead(ctx, newBlock, beaconState2); err != nil {
		t.Fatal(err)
	}

	// At this point, once the two nodes are fully caught up, we expect their state,
	// in particular their balances, to be equal.
	// We look at chain heads.
	nodeAHead, err = chainService1.beaconDB.ChainHead()
	if err != nil {
		t.Fatal(err)
	}
	nodeBHead, err = chainService2.beaconDB.ChainHead()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Node A head slot: %v", nodeAHead.Slot-params.BeaconConfig().GenesisSlot)
	t.Logf("Node B head slot: %v", nodeBHead.Slot-params.BeaconConfig().GenesisSlot)

	// We then compare their balances.
	finalState1, err := chainService1.beaconDB.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	finalState2, err := chainService2.beaconDB.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	balancesNodeA = finalState1.ValidatorBalances
	balancesNodeB = finalState2.ValidatorBalances
	for i := range balancesNodeA {
		if balancesNodeA[i] != balancesNodeB[i] {
			t.Errorf(
				"Expected balance to match at index %d, received %v = %v",
				i,
				balancesNodeA[i],
				balancesNodeB[i],
			)
		}
	}
}
