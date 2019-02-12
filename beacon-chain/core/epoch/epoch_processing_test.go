package epoch

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/ssz"
)

func TestCanProcessEpoch(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	tests := []struct {
		slot            uint64
		canProcessEpoch bool
	}{
		{
			slot:            1,
			canProcessEpoch: false,
		},
		{
			slot:            63,
			canProcessEpoch: false,
		},
		{
			slot:            64,
			canProcessEpoch: true,
		}, {
			slot:            128,
			canProcessEpoch: true,
		}, {
			slot:            1000000000,
			canProcessEpoch: true,
		},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if CanProcessEpoch(state) != tt.canProcessEpoch {
			t.Errorf(
				"CanProcessEpoch(%d) = %v. Wanted %v",
				tt.slot,
				CanProcessEpoch(state),
				tt.canProcessEpoch,
			)
		}
	}
}

func TestCanProcessEth1Data(t *testing.T) {
	if params.BeaconConfig().Eth1DataVotingPeriod != 16 {
		t.Errorf("Eth1DataVotingPeriod should be 16 for these tests to pass")
	}
	tests := []struct {
		slot               uint64
		canProcessEth1Data bool
	}{
		{
			slot:               1,
			canProcessEth1Data: false,
		},
		{
			slot:               15,
			canProcessEth1Data: false,
		},
		{
			slot:               15 * params.BeaconConfig().EpochLength,
			canProcessEth1Data: true,
		},
		{
			slot:               127 * params.BeaconConfig().EpochLength,
			canProcessEth1Data: true,
		},
		{
			slot:               234234,
			canProcessEth1Data: false,
		},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{Slot: tt.slot}
		if CanProcessEth1Data(state) != tt.canProcessEth1Data {
			t.Errorf(
				"CanProcessEth1Data(%d) = %v. Wanted %v",
				tt.slot,
				CanProcessEth1Data(state),
				tt.canProcessEth1Data,
			)
		}
	}
}

func TestProcessEth1Data(t *testing.T) {
	requiredVoteCount := params.BeaconConfig().Eth1DataVotingPeriod
	state := &pb.BeaconState{
		Slot: 15 * params.BeaconConfig().EpochLength,
		LatestEth1Data: &pb.Eth1Data{
			DepositRootHash32: nil,
			BlockHash32:       nil,
		},
		Eth1DataVotes: []*pb.Eth1DataVote{
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'A'},
					BlockHash32:       []byte{'B'},
				},
				VoteCount: 0,
			},
			// DepositRootHash32 ['B'] gets to process with sufficient vote count.
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'C'},
					BlockHash32:       []byte{'D'},
				},
				VoteCount: requiredVoteCount/2 + 1,
			},
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'E'},
					BlockHash32:       []byte{'F'},
				},
				VoteCount: requiredVoteCount / 2,
			},
		},
	}
	newState := ProcessEth1Data(state)
	if !bytes.Equal(newState.LatestEth1Data.DepositRootHash32, []byte{'C'}) {
		t.Errorf("Incorrect DepositRootHash32. Wanted: %v, got: %v",
			[]byte{'C'}, newState.LatestEth1Data.DepositRootHash32)
	}

	// Adding a new receipt root ['D'] which should be the new processed receipt root.
	state.Eth1DataVotes = append(state.Eth1DataVotes,
		&pb.Eth1DataVote{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{'G'},
				BlockHash32:       []byte{'H'},
			},
			VoteCount: requiredVoteCount,
		},
	)
	newState = ProcessEth1Data(state)
	if !bytes.Equal(newState.LatestEth1Data.DepositRootHash32, []byte{'G'}) {
		t.Errorf("Incorrect DepositRootHash32. Wanted: %v, got: %v",
			[]byte{'G'}, newState.LatestEth1Data.DepositRootHash32)
	}

	if len(newState.Eth1DataVotes) != 0 {
		t.Errorf("Failed to clean up Eth1DataVotes slice. Length: %d",
			len(newState.Eth1DataVotes))
	}
}

func TestProcessEth1Data_InactionSlot(t *testing.T) {
	requiredVoteCount := params.BeaconConfig().Eth1DataVotingPeriod
	state := &pb.BeaconState{
		Slot: 4,
		LatestEth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte{'A'},
			BlockHash32:       []byte{'B'},
		},
		Eth1DataVotes: []*pb.Eth1DataVote{
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'C'},
					BlockHash32:       []byte{'D'},
				},
				VoteCount: requiredVoteCount/2 + 1,
			},
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'E'},
					BlockHash32:       []byte{'F'},
				},
				VoteCount: requiredVoteCount / 2,
			},
			{
				Eth1Data: &pb.Eth1Data{
					DepositRootHash32: []byte{'G'},
					BlockHash32:       []byte{'H'},
				},
				VoteCount: requiredVoteCount,
			},
		},
	}

	// Adding a new receipt root ['D'] which should be the new processed receipt root.
	newState := ProcessEth1Data(state)
	if !bytes.Equal(newState.LatestEth1Data.DepositRootHash32, []byte{'A'}) {
		t.Errorf("Incorrect DepositRootHash32. Wanted: %v, got: %v",
			[]byte{'A'}, newState.LatestEth1Data.DepositRootHash32)
	}
}

func TestProcessJustification(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	state := &pb.BeaconState{
		Slot:                  300,
		JustifiedEpoch:        3,
		JustificationBitfield: 4,
	}
	newState := ProcessJustification(state, 1, 1, 1)

	if newState.PreviousJustifiedEpoch != 3 {
		t.Errorf("New state's prev justified slot %d != old state's justified slot %d",
			newState.PreviousJustifiedEpoch, state.JustifiedEpoch)
	}
	// Since this epoch was justified (not prev), justified_epoch = slot_to_epoch(state.slot) -1.
	if newState.JustifiedEpoch != helpers.PrevEpoch(state) {
		t.Errorf("New state's justified epoch %d != state's slot - EPOCH_LENGTH %d",
			newState.JustifiedEpoch, helpers.PrevEpoch(state))
	}
	// The new JustificationBitfield is 11, it went from 0100 to 1011. Two 1's were appended because both
	// prev epoch and this epoch were justified.
	if newState.JustificationBitfield != 11 {
		t.Errorf("New state's justification bitfield %d != 11", newState.JustificationBitfield)
	}

	// Assume for the case where only prev epoch got justified. Verify
	// justified_epoch = slot_to_epoch(state.slot) -2.
	newState = ProcessJustification(state, 0, 1, 1)
	if newState.JustifiedEpoch != helpers.PrevEpoch(state)-1 {
		t.Errorf("New state's justified epoch %d != state's epoch -2 %d",
			newState.JustifiedEpoch, helpers.PrevEpoch(state)-1)
	}
}

func TestProcessFinalization(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	// 2 consecutive justified epoch in a row,
	// and previous justified epoch is slot_to_epoch(state.slot) - 2.
	state := &pb.BeaconState{
		Slot:                   200,
		JustifiedEpoch:         2,
		PreviousJustifiedEpoch: 1,
		JustificationBitfield:  3,
	}
	newState := ProcessFinalization(state)
	if newState.FinalizedEpoch != state.JustifiedEpoch {
		t.Errorf("Wanted finalized epoch to be %d, got %d:",
			state.JustifiedEpoch, newState.FinalizedEpoch)
	}

	// 3 consecutive justified epoch in a row,
	// and previous justified epoch is slot_to_epoch(state.slot) - 3.
	state = &pb.BeaconState{
		Slot:                   300,
		JustifiedEpoch:         3,
		PreviousJustifiedEpoch: 1,
		JustificationBitfield:  7,
	}
	newState = ProcessFinalization(state)
	if newState.FinalizedEpoch != state.JustifiedEpoch {
		t.Errorf("Wanted finalized epoch to be %d, got %d:",
			state.JustifiedEpoch, newState.FinalizedEpoch)
	}

	// 4 consecutive justified epoch in a row,
	// and previous justified epoch is slot_to_epoch(state.slot) - 3.
	state = &pb.BeaconState{
		Slot:                   400,
		JustifiedEpoch:         5,
		PreviousJustifiedEpoch: 2,
		JustificationBitfield:  15,
	}
	newState = ProcessFinalization(state)
	if newState.FinalizedEpoch != state.JustifiedEpoch {
		t.Errorf("Wanted finalized epoch to be %d, got %d:",
			state.JustifiedEpoch, newState.FinalizedEpoch)
	}

	// if nothing gets finalized it just returns the same state.
	state = &pb.BeaconState{
		Slot:                   100,
		JustifiedEpoch:         1,
		PreviousJustifiedEpoch: 0,
		JustificationBitfield:  1,
	}
	newState = ProcessFinalization(state)
	if newState.FinalizedEpoch != 0 {
		t.Errorf("Wanted finalized epoch to be %d, got %d:",
			0, newState.FinalizedEpoch)
	}
}

func TestProcessCrosslinksOk(t *testing.T) {
	state := buildState(5, params.BeaconConfig().DepositsForChainStart)
	state.LatestCrosslinks = []*pb.Crosslink{{}, {}}
	epoch := uint64(5)
	state.Slot = epoch * params.BeaconConfig().EpochLength

	byteLength := int(params.BeaconConfig().DepositsForChainStart / params.BeaconConfig().TargetCommitteeSize / 8)
	var participationBitfield []byte
	for i := 0; i < byteLength; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                 state.Slot,
				ShardBlockRootHash32: []byte{'A'},
			},
			// All validators attested to the above roots.
			AggregationBitfield: participationBitfield,
		}
		attestations = append(attestations, attestation)
	}

	newState, err := ProcessCrosslinks(
		state,
		attestations,
		nil,
	)
	if err != nil {
		t.Fatalf("Could not execute ProcessCrosslinks: %v", err)
	}
	// Verify crosslink for shard 0([1]) was processed at state.slot (5).
	if newState.LatestCrosslinks[0].Epoch != epoch {
		t.Errorf("Shard 0s got crosslinked at epoch %d, wanted: %d",
			newState.LatestCrosslinks[0].Epoch, epoch)
	}
	// Verify crosslink for shard 0 was root hashed for []byte{'A'}.
	if !bytes.Equal(newState.LatestCrosslinks[0].ShardBlockRootHash32,
		attestations[0].Data.ShardBlockRootHash32) {
		t.Errorf("Shard 0's root hash is %#x, wanted: %#x",
			newState.LatestCrosslinks[0].ShardBlockRootHash32,
			attestations[0].Data.ShardBlockRootHash32)
	}
}

func TestProcessCrosslinksNoParticipantsBitField(t *testing.T) {
	state := buildState(5, params.BeaconConfig().DepositsForChainStart)
	state.LatestCrosslinks = []*pb.Crosslink{{}, {}}

	attestations := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{},
			// Empty participation bitfield will trigger error.
			AggregationBitfield: []byte{}}}

	wanted := fmt.Sprintf(
		"wanted participants bitfield length %d, got: %d",
		16, 0,
	)
	if _, err := ProcessCrosslinks(state, attestations, nil); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestProcessEjectionsOk(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 1,
		ValidatorBalances: []uint64{
			params.BeaconConfig().EjectionBalance - 1,
			params.BeaconConfig().EjectionBalance + 1},
		LatestPenalizedBalances: []uint64{0},
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch}},
	}

	state, err := ProcessEjections(state)
	if err != nil {
		t.Fatalf("Could not execute ProcessEjections: %v", err)
	}

	if state.ValidatorRegistry[0].ExitEpoch !=
		params.BeaconConfig().EntryExitDelay+state.Slot {
		t.Errorf("Expected exit epoch %d, but got %d",
			state.ValidatorRegistry[0].ExitEpoch, params.BeaconConfig().EntryExitDelay)
	}
	if state.ValidatorRegistry[1].ExitEpoch !=
		params.BeaconConfig().FarFutureEpoch {
		t.Errorf("Expected exit epoch 0, but got %v", state.ValidatorRegistry[1].ExitEpoch)
	}
}

func TestCanProcessValidatorRegistry(t *testing.T) {
	crosslinks := make([]*pb.Crosslink, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &pb.Crosslink{
			Epoch: 101,
		}
	}

	state := &pb.BeaconState{
		FinalizedEpoch:               1,
		ValidatorRegistryUpdateEpoch: 0,
		LatestCrosslinks:             crosslinks,
	}

	if !CanProcessValidatorRegistry(state) {
		t.Errorf("Wanted True for CanProcessValidatorRegistry, but got %v", CanProcessValidatorRegistry(state))
	}
}

func TestCanNotProcessValidatorRegistry(t *testing.T) {
	state := &pb.BeaconState{
		FinalizedEpoch:               1,
		ValidatorRegistryUpdateEpoch: 101,
	}

	if CanProcessValidatorRegistry(state) {
		t.Errorf("Wanted False for CanProcessValidatorRegistry, but got %v", CanProcessValidatorRegistry(state))
	}
	state = &pb.BeaconState{
		ValidatorRegistryUpdateEpoch: 101,
		FinalizedEpoch:               1,
		LatestCrosslinks: []*pb.Crosslink{
			{Epoch: 100},
		},
	}
	if CanProcessValidatorRegistry(state) {
		t.Errorf("Wanted False for CanProcessValidatorRegistry, but got %v", CanProcessValidatorRegistry(state))
	}
}

func TestProcessPrevSlotShardOk(t *testing.T) {
	state := &pb.BeaconState{
		CurrentCalculationEpoch: 1,
		CurrentEpochStartShard:  2,
		CurrentEpochSeedHash32:  []byte{'A'},
	}

	newState := ProcessPrevSlotShardSeed(
		proto.Clone(state).(*pb.BeaconState))

	if newState.PreviousCalculationEpoch != state.CurrentCalculationEpoch {
		t.Errorf("Incorret prev epoch calculation slot: Wanted: %d, got: %d",
			newState.PreviousCalculationEpoch, state.CurrentCalculationEpoch)
	}
	if newState.PreviousEpochStartShard != state.CurrentEpochStartShard {
		t.Errorf("Incorret prev epoch start shard: Wanted: %d, got: %d",
			newState.PreviousEpochStartShard, state.CurrentEpochStartShard)
	}
	if !bytes.Equal(newState.PreviousEpochSeedHash32, state.CurrentEpochSeedHash32) {
		t.Errorf("Incorret prev epoch seed mix hash: Wanted: %v, got: %v",
			state.CurrentEpochSeedHash32, newState.PreviousEpochSeedHash32)
	}
}

func TestProcessValidatorRegistryOk(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                     params.BeaconConfig().SeedLookahead,
		LatestRandaoMixesHash32S: [][]byte{{'A'}, {'B'}},
		CurrentEpochSeedHash32:   []byte{'C'},
	}
	newState, err := ProcessValidatorRegistry(
		proto.Clone(state).(*pb.BeaconState))
	if err != nil {
		t.Fatalf("Could not execute ProcessValidatorRegistry: %v", err)
	}
	if newState.CurrentCalculationEpoch != state.Slot {
		t.Errorf("Incorret curr epoch calculation slot: Wanted: %d, got: %d",
			newState.CurrentCalculationEpoch, state.Slot)
	}
	if !bytes.Equal(newState.CurrentEpochSeedHash32, state.LatestRandaoMixesHash32S[0]) {
		t.Errorf("Incorret current epoch seed mix hash: Wanted: %v, got: %v",
			state.LatestRandaoMixesHash32S[0], newState.CurrentEpochSeedHash32)
	}
}

func TestProcessPartialValidatorRegistry(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                     params.BeaconConfig().EpochLength * 2,
		LatestRandaoMixesHash32S: [][]byte{{'A'}, {'B'}, {'C'}},
		LatestIndexRootHash32S:   [][]byte{{'D'}, {'E'}, {'F'}},
	}
	copiedState := proto.Clone(state).(*pb.BeaconState)
	newState, err := ProcessPartialValidatorRegistry(copiedState)
	if err != nil {
		t.Fatalf("could not ProcessPartialValidatorRegistry: %v", err)
	}
	if newState.CurrentCalculationEpoch != helpers.NextEpoch(state) {
		t.Errorf("Incorrect CurrentCalculationEpoch, wanted: %d, got: %d",
			helpers.NextEpoch(state), newState.CurrentCalculationEpoch)
	}
}

func TestCleanupAttestations(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	epochLength := params.BeaconConfig().EpochLength
	state := &pb.BeaconState{
		Slot: epochLength,
		LatestAttestations: []*pb.PendingAttestation{
			{Data: &pb.AttestationData{Slot: 1}},
			{Data: &pb.AttestationData{Slot: epochLength - 10}},
			{Data: &pb.AttestationData{Slot: epochLength}},
			{Data: &pb.AttestationData{Slot: epochLength + 1}},
			{Data: &pb.AttestationData{Slot: epochLength + 20}},
			{Data: &pb.AttestationData{Slot: 32}},
			{Data: &pb.AttestationData{Slot: 33}},
			{Data: &pb.AttestationData{Slot: 2 * epochLength}},
		},
	}
	wanted := &pb.BeaconState{
		Slot: epochLength,
		LatestAttestations: []*pb.PendingAttestation{
			{Data: &pb.AttestationData{Slot: epochLength}},
			{Data: &pb.AttestationData{Slot: epochLength + 1}},
			{Data: &pb.AttestationData{Slot: epochLength + 20}},
			{Data: &pb.AttestationData{Slot: 2 * epochLength}},
		},
	}
	newState := CleanupAttestations(state)

	if !reflect.DeepEqual(newState, wanted) {
		t.Errorf("Wanted state: %v, got state: %v ",
			wanted, newState)
	}
}

func TestUpdateLatestPenalizedBalances_Ok(t *testing.T) {
	tests := []struct {
		epoch    uint64
		balances uint64
	}{
		{
			epoch:    0,
			balances: 100,
		},
		{
			epoch:    params.BeaconConfig().LatestPenalizedExitLength,
			balances: 324,
		},
		{
			epoch:    params.BeaconConfig().LatestPenalizedExitLength + 1,
			balances: 234324,
		}, {
			epoch:    params.BeaconConfig().LatestPenalizedExitLength * 100,
			balances: 34,
		}, {
			epoch:    params.BeaconConfig().LatestPenalizedExitLength * 1000,
			balances: 1,
		},
	}
	for _, tt := range tests {
		epoch := tt.epoch % params.BeaconConfig().LatestPenalizedExitLength
		latestPenalizedExitBalances := make([]uint64,
			params.BeaconConfig().LatestPenalizedExitLength)
		latestPenalizedExitBalances[epoch] = tt.balances
		state := &pb.BeaconState{
			Slot:                    tt.epoch * params.BeaconConfig().EpochLength,
			LatestPenalizedBalances: latestPenalizedExitBalances}
		newState := UpdateLatestPenalizedBalances(state)
		if newState.LatestPenalizedBalances[epoch+1] !=
			tt.balances {
			t.Errorf(
				"LatestPenalizedBalances didn't update for epoch %d,"+
					"wanted: %d, got: %d", epoch+1, tt.balances,
				newState.LatestPenalizedBalances[epoch+1],
			)
		}
	}
}

func TestUpdateLatestRandaoMixes_Ok(t *testing.T) {
	tests := []struct {
		epoch uint64
		seed  []byte
	}{
		{
			epoch: 0,
			seed:  []byte{'A'},
		},
		{
			epoch: 1,
			seed:  []byte{'B'},
		},
		{
			epoch: 100,
			seed:  []byte{'C'},
		}, {
			epoch: params.BeaconConfig().LatestRandaoMixesLength * 100,
			seed:  []byte{'D'},
		}, {
			epoch: params.BeaconConfig().LatestRandaoMixesLength * 1000,
			seed:  []byte{'E'},
		},
	}
	for _, tt := range tests {
		epoch := tt.epoch % params.BeaconConfig().LatestRandaoMixesLength
		latestPenalizedRandaoMixes := make([][]byte,
			params.BeaconConfig().LatestRandaoMixesLength)
		latestPenalizedRandaoMixes[epoch] = tt.seed
		state := &pb.BeaconState{
			Slot:                     tt.epoch * params.BeaconConfig().EpochLength,
			LatestRandaoMixesHash32S: latestPenalizedRandaoMixes}
		newState, err := UpdateLatestRandaoMixes(state)
		if err != nil {
			t.Fatalf("could not update latest randao mixes: %v", err)
		}
		if !bytes.Equal(newState.LatestRandaoMixesHash32S[epoch+1], tt.seed) {
			t.Errorf(
				"LatestRandaoMixes didn't update for epoch %d,"+
					"wanted: %v, got: %v", epoch+1, tt.seed,
				newState.LatestRandaoMixesHash32S[epoch+1],
			)
		}
	}
}

func TestUpdateLatestIndexRoots_Ok(t *testing.T) {
	epoch := uint64(1234)
	latestIndexRoots := make([][]byte,
		params.BeaconConfig().LatestIndexRootsLength)
	state := &pb.BeaconState{
		Slot:                   epoch * params.BeaconConfig().EpochLength,
		LatestIndexRootHash32S: latestIndexRoots}
	newState, err := UpdateLatestIndexRoots(state)
	if err != nil {
		t.Fatalf("could not update latest index roots: %v", err)
	}
	nextEpoch := helpers.NextEpoch(state) + params.BeaconConfig().EntryExitDelay
	indexRoot, err := ssz.TreeHash(helpers.ActiveValidatorIndices(state.ValidatorRegistry, nextEpoch))
	if err != nil {
		t.Fatalf("could not ssz index root: %v", err)
	}
	if !bytes.Equal(newState.LatestIndexRootHash32S[nextEpoch], indexRoot[:]) {
		t.Errorf(
			"LatestIndexRootHash32S didn't update for epoch %d,"+
				"wanted: %v, got: %v", nextEpoch, indexRoot,
			newState.LatestIndexRootHash32S[nextEpoch],
		)
	}
}
