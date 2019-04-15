package blockchain

import (
	"context"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"testing"
	"time"
)

type assignment struct {
	shard uint64
	validatorIndex uint64
	committee []uint64
}

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

	chainService1 := setupBeaconChain(t, beaconDB1, nil)
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
	genesisState, err := beaconDB1.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Then, we create the chain up to slot 10 in both.
	beaconState := genesisState
    blocks := []*pb.BeaconBlock{genesisBlock}
    assignments := make(map[uint64]*assignment)
    for idx := range genesisState.ValidatorRegistry {
		committee, shard, slot, _, err :=
			helpers.CommitteeAssignment(genesisState, genesisBlock.Slot, uint64(idx), false)
		if err != nil {
            t.Fatal(err)
		}
		assignments[slot-params.BeaconConfig().GenesisSlot] = &assignment{
			shard,
			uint64(idx),
			committee,
		}
	}
	for i := 1; i <= 10; i++ {
		parent := blocks[i-1]
		prevBlockRoot, err := hashutil.HashBeaconBlock(parent)
		if err != nil {
			t.Fatal(err)
		}
		block := &pb.BeaconBlock{
			Slot:             uint64(i),
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
		prevSlot := uint64(i-1)
		committee := assignments[prevSlot].committee
		shard := assignments[prevSlot].shard
		attestation := &pb.Attestation{}
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
			epochBoundaryRoot, err = b.BlockRoot(beaconState, epochStartSlot)
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
		justifiedBlockRoot := beaconState.JustifiedRoot

		// If an attester has to attest for genesis block.
		if beaconState.Slot == params.BeaconConfig().GenesisSlot {
			epochBoundaryRoot = params.BeaconConfig().ZeroHash[:]
			justifiedBlockRoot = params.BeaconConfig().ZeroHash[:]
		}
		attestation.Data.Slot = prevSlot
		attestation.Data.Shard = shard
		attestation.Data.EpochBoundaryRootHash32 = epochBoundaryRoot
		attestation.Data.JustifiedBlockRootHash32 = justifiedBlockRoot
		attestation.Data.JustifiedEpoch = beaconState.JustifiedEpoch
		attestation.Data.LatestCrosslink = beaconState.LatestCrosslinks[shard]

		block.Body.Attestations = []*pb.Attestation{attestation}
		blocks[i] = block
		beaconState, err = chainService1.ApplyBlockStateTransition(ctx, block, beaconState)
		if err != nil {
			t.Fatal(err)
		}
	}

	// We update attestation targets for node A such that validators point to the block
	// at slot 7 as canonical - then, a reorg to that slot will occur.

	// We then proceed in both nodes normally through several blocks.

	// At this point, once the two nodes are fully caught up, we expect their state,
	// in particular their balances, to be equal.
}
