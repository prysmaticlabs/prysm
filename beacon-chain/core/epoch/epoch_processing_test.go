package epoch

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: true,
	})
}

func TestCanProcessEpoch_TrueOnEpochs(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	tests := []struct {
		slot            uint64
		canProcessEpoch bool
	}{
		{
			slot:            1,
			canProcessEpoch: false,
		}, {
			slot:            63,
			canProcessEpoch: true,
		},
		{
			slot:            64,
			canProcessEpoch: false,
		}, {
			slot:            127,
			canProcessEpoch: true,
		}, {
			slot:            1000000000,
			canProcessEpoch: false,
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

func TestCanProcessEth1Data_TrueOnVotingPeriods(t *testing.T) {
	if params.BeaconConfig().EpochsPerEth1VotingPeriod != 16 {
		t.Errorf("EpochsPerEth1VotingPeriodshould be 16 for these tests to pass")
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
			slot:               15 * params.BeaconConfig().SlotsPerEpoch,
			canProcessEth1Data: true,
		},
		{
			slot:               127 * params.BeaconConfig().SlotsPerEpoch,
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

func TestProcessEth1Data_UpdatesStateAndCleans(t *testing.T) {
	requiredVoteCount := params.BeaconConfig().EpochsPerEth1VotingPeriod *
		params.BeaconConfig().SlotsPerEpoch
	state := &pb.BeaconState{
		Slot: 15 * params.BeaconConfig().SlotsPerEpoch,
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
	requiredVoteCount := params.BeaconConfig().EpochsPerEth1VotingPeriod
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

func TestProcessJustification_PreviousEpochJustified(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	var latestBlockRoots [][]byte

	for i := uint64(0); i < params.BeaconConfig().LatestBlockRootsLength; i++ {
		latestBlockRoots = append(latestBlockRoots, []byte("a"))
	}

	state := &pb.BeaconState{
		Slot:                   300 + params.BeaconConfig().GenesisSlot,
		JustifiedEpoch:         3,
		JustificationBitfield:  4,
		LatestBlockRootHash32S: latestBlockRoots,
	}
	newState, err := ProcessJustificationAndFinalization(
		state,
		1,
		1,
		1,
		1,
	)
	if err != nil {
		t.Errorf("Could not process justification and finalization of state %v", err)
	}

	if newState.PreviousJustifiedEpoch != 3 {
		t.Errorf("New state's prev justified slot %d != old state's justified slot %d",
			newState.PreviousJustifiedEpoch, state.JustifiedEpoch)
	}
	// Since this epoch was justified (not prev), justified_epoch = slot_to_epoch(state.slot) -1.
	if newState.JustifiedEpoch != helpers.CurrentEpoch(state) {
		t.Errorf("New state's justified epoch %d != state's slot - SLOTS_PER_EPOCH: %d",
			newState.JustifiedEpoch, helpers.CurrentEpoch(state))
	}
	// The new JustificationBitfield is 11, it went from 0100 to 1011. Two 1's were appended because both
	// prev epoch and this epoch were justified.
	if newState.JustificationBitfield != 11 {
		t.Errorf("New state's justification bitfield %d != 11", newState.JustificationBitfield)
	}

	// Assume for the case where only prev epoch got justified. Verify
	// justified_epoch = slot_to_epoch(state.slot) -2.
	newState, err = ProcessJustificationAndFinalization(
		state,
		0,
		1,
		1,
		1,
	)
	if err != nil {
		t.Errorf("Could not process justification and finalization of state %v", err)
	}
	if newState.JustifiedEpoch != helpers.CurrentEpoch(state)-1 {
		t.Errorf("New state's justified epoch %d != state's epoch -2: %d",
			newState.JustifiedEpoch, helpers.CurrentEpoch(state)-1)
	}
}

func TestProcessCrosslinks_CrosslinksCorrectEpoch(t *testing.T) {
	state := buildState(5, params.BeaconConfig().DepositsForChainStart)
	state.LatestCrosslinks = []*pb.Crosslink{{}, {}}
	epoch := uint64(5)
	state.Slot = params.BeaconConfig().GenesisSlot + epoch*params.BeaconConfig().SlotsPerEpoch

	byteLength := int(params.BeaconConfig().DepositsForChainStart / params.BeaconConfig().TargetCommitteeSize / 8)
	var participationBitfield []byte
	for i := 0; i < byteLength; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                    state.Slot,
				CrosslinkDataRootHash32: []byte{'A'},
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
	// Verify crosslink for shard 0([1]) was processed at genesis epoch + 5.
	if newState.LatestCrosslinks[0].Epoch != params.BeaconConfig().GenesisEpoch+epoch {
		t.Errorf("Shard 0s got crosslinked at epoch %d, wanted: %d",
			newState.LatestCrosslinks[0].Epoch, +params.BeaconConfig().GenesisSlot)
	}
	// Verify crosslink for shard 0 was root hashed for []byte{'A'}.
	if !bytes.Equal(newState.LatestCrosslinks[0].CrosslinkDataRootHash32,
		attestations[0].Data.CrosslinkDataRootHash32) {
		t.Errorf("Shard 0's root hash is %#x, wanted: %#x",
			newState.LatestCrosslinks[0].CrosslinkDataRootHash32,
			attestations[0].Data.CrosslinkDataRootHash32)
	}
}

func TestProcessCrosslinks_NoParticipantsBitField(t *testing.T) {
	state := buildState(params.BeaconConfig().GenesisSlot+5, params.BeaconConfig().DepositsForChainStart)
	state.LatestCrosslinks = []*pb.Crosslink{{}, {}}

	attestations := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
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

func TestProcessEjections_EjectsAtCorrectSlot(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 1,
		Balances: []uint64{
			params.BeaconConfig().EjectionBalance - 1,
			params.BeaconConfig().EjectionBalance + 1},
		LatestSlashedBalances: []uint64{0},
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch}},
	}

	state, err := ProcessEjections(state, false /* disable logging */)
	if err != nil {
		t.Fatalf("Could not execute ProcessEjections: %v", err)
	}

	if state.ValidatorRegistry[0].ExitEpoch !=
		params.BeaconConfig().ActivationExitDelay+state.Slot {
		t.Errorf("Expected exit epoch %d, but got %d",
			state.ValidatorRegistry[0].ExitEpoch, params.BeaconConfig().ActivationExitDelay)
	}
	if state.ValidatorRegistry[1].ExitEpoch !=
		params.BeaconConfig().FarFutureEpoch {
		t.Errorf("Expected exit epoch 0, but got %v", state.ValidatorRegistry[1].ExitEpoch)
	}
}

func TestCanProcessValidatorRegistry_OnFarEpoch(t *testing.T) {
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

	if processed := CanProcessValidatorRegistry(state); !processed {
		t.Errorf("Wanted True for CanProcessValidatorRegistry, but got %v", processed)
	}
}

func TestCanProcessValidatorRegistry_OutOfBounds(t *testing.T) {
	state := &pb.BeaconState{
		FinalizedEpoch:               1,
		ValidatorRegistryUpdateEpoch: 101,
	}

	if processed := CanProcessValidatorRegistry(state); processed {
		t.Errorf("Wanted False for CanProcessValidatorRegistry, but got %v", processed)
	}
	state = &pb.BeaconState{
		ValidatorRegistryUpdateEpoch: 101,
		FinalizedEpoch:               1,
		LatestCrosslinks: []*pb.Crosslink{
			{Epoch: 100},
		},
	}
	if processed := CanProcessValidatorRegistry(state); processed {
		t.Errorf("Wanted False for CanProcessValidatorRegistry, but got %v", processed)
	}
}

func TestProcessPrevSlotShard_CorrectPrevEpochData(t *testing.T) {
	state := &pb.BeaconState{
		CurrentShufflingEpoch:      1,
		CurrentShufflingStartShard: 2,
		CurrentShufflingSeedHash32: []byte{'A'},
	}

	newState := ProcessPrevSlotShardSeed(
		proto.Clone(state).(*pb.BeaconState))

	if newState.PreviousShufflingEpoch != state.CurrentShufflingEpoch {
		t.Errorf("Incorrect prev epoch calculation slot: Wanted: %d, got: %d",
			newState.PreviousShufflingEpoch, state.CurrentShufflingEpoch)
	}
	if newState.PreviousShufflingStartShard != state.CurrentShufflingStartShard {
		t.Errorf("Incorrect prev epoch start shard: Wanted: %d, got: %d",
			newState.PreviousShufflingStartShard, state.CurrentShufflingStartShard)
	}
	if !bytes.Equal(newState.PreviousShufflingSeedHash32, state.CurrentShufflingSeedHash32) {
		t.Errorf("Incorrect prev epoch seed mix hash: Wanted: %v, got: %v",
			state.CurrentShufflingSeedHash32, newState.PreviousShufflingSeedHash32)
	}
}

func TestProcessPartialValidatorRegistry_CorrectShufflingEpoch(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch * 2,
		LatestRandaoMixes:      [][]byte{{'A'}, {'B'}, {'C'}},
		LatestIndexRootHash32S: [][]byte{{'D'}, {'E'}, {'F'}},
	}
	copiedState := proto.Clone(state).(*pb.BeaconState)
	newState, err := ProcessPartialValidatorRegistry(copiedState)
	if err != nil {
		t.Fatalf("could not ProcessPartialValidatorRegistry: %v", err)
	}
	if newState.CurrentShufflingEpoch != helpers.NextEpoch(state) {
		t.Errorf("Incorrect CurrentShufflingEpoch, wanted: %d, got: %d",
			helpers.NextEpoch(state), newState.CurrentShufflingEpoch)
	}
}

func TestCleanupAttestations_RemovesFromLastEpoch(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	state := &pb.BeaconState{
		Slot: slotsPerEpoch,
		LatestAttestations: []*pb.PendingAttestation{
			{Data: &pb.AttestationData{Slot: 1}},
			{Data: &pb.AttestationData{Slot: slotsPerEpoch - 10}},
			{Data: &pb.AttestationData{Slot: slotsPerEpoch}},
			{Data: &pb.AttestationData{Slot: slotsPerEpoch + 1}},
			{Data: &pb.AttestationData{Slot: slotsPerEpoch + 20}},
			{Data: &pb.AttestationData{Slot: 32}},
			{Data: &pb.AttestationData{Slot: 33}},
			{Data: &pb.AttestationData{Slot: 2 * slotsPerEpoch}},
		},
	}
	wanted := &pb.BeaconState{
		Slot: slotsPerEpoch,
		LatestAttestations: []*pb.PendingAttestation{
			{Data: &pb.AttestationData{Slot: slotsPerEpoch}},
			{Data: &pb.AttestationData{Slot: slotsPerEpoch + 1}},
			{Data: &pb.AttestationData{Slot: slotsPerEpoch + 20}},
			{Data: &pb.AttestationData{Slot: 2 * slotsPerEpoch}},
		},
	}
	newState := CleanupAttestations(state)

	if !reflect.DeepEqual(newState, wanted) {
		t.Errorf("Wanted state: %v, got state: %v ",
			wanted, newState)
	}
}

func TestUpdateLatestSlashedBalances_UpdatesBalances(t *testing.T) {
	tests := []struct {
		epoch    uint64
		balances uint64
	}{
		{
			epoch:    0,
			balances: 100,
		},
		{
			epoch:    params.BeaconConfig().LatestSlashedExitLength,
			balances: 324,
		},
		{
			epoch:    params.BeaconConfig().LatestSlashedExitLength + 1,
			balances: 234324,
		}, {
			epoch:    params.BeaconConfig().LatestSlashedExitLength * 100,
			balances: 34,
		}, {
			epoch:    params.BeaconConfig().LatestSlashedExitLength * 1000,
			balances: 1,
		},
	}
	for _, tt := range tests {
		epoch := tt.epoch % params.BeaconConfig().LatestSlashedExitLength
		latestSlashedExitBalances := make([]uint64,
			params.BeaconConfig().LatestSlashedExitLength)
		latestSlashedExitBalances[epoch] = tt.balances
		state := &pb.BeaconState{
			Slot:                  tt.epoch * params.BeaconConfig().SlotsPerEpoch,
			LatestSlashedBalances: latestSlashedExitBalances}
		newState := UpdateLatestSlashedBalances(state)
		if newState.LatestSlashedBalances[epoch+1] !=
			tt.balances {
			t.Errorf(
				"LatestSlashedBalances didn't update for epoch %d,"+
					"wanted: %d, got: %d", epoch+1, tt.balances,
				newState.LatestSlashedBalances[epoch+1],
			)
		}
	}
}

func TestUpdateLatestRandaoMixes_UpdatesRandao(t *testing.T) {
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
		latestSlashedRandaoMixes := make([][]byte,
			params.BeaconConfig().LatestRandaoMixesLength)
		latestSlashedRandaoMixes[epoch] = tt.seed
		state := &pb.BeaconState{
			Slot:              tt.epoch * params.BeaconConfig().SlotsPerEpoch,
			LatestRandaoMixes: latestSlashedRandaoMixes}
		newState, err := UpdateLatestRandaoMixes(state)
		if err != nil {
			t.Fatalf("could not update latest randao mixes: %v", err)
		}
		if !bytes.Equal(newState.LatestRandaoMixes[epoch+1], tt.seed) {
			t.Errorf(
				"LatestRandaoMixes didn't update for epoch %d,"+
					"wanted: %v, got: %v", epoch+1, tt.seed,
				newState.LatestRandaoMixes[epoch+1],
			)
		}
	}
}

func TestUpdateLatestActiveIndexRoots_UpdatesActiveIndexRoots(t *testing.T) {
	epoch := uint64(1234)
	latestActiveIndexRoots := make([][]byte,
		params.BeaconConfig().LatestActiveIndexRootsLength)
	state := &pb.BeaconState{
		Slot:                   epoch * params.BeaconConfig().SlotsPerEpoch,
		LatestIndexRootHash32S: latestActiveIndexRoots}
	newState, err := UpdateLatestActiveIndexRoots(state)
	if err != nil {
		t.Fatalf("could not update latest index roots: %v", err)
	}
	nextEpoch := helpers.NextEpoch(state) + params.BeaconConfig().ActivationExitDelay
	validatorIndices := helpers.ActiveValidatorIndices(state, nextEpoch)
	indicesBytes := []byte{}
	for _, val := range validatorIndices {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, val)
		indicesBytes = append(indicesBytes, buf...)
	}
	indexRoot := hashutil.Hash(indicesBytes)
	if !bytes.Equal(newState.LatestIndexRootHash32S[nextEpoch], indexRoot[:]) {
		t.Errorf(
			"LatestIndexRootHash32S didn't update for epoch %d,"+
				"wanted: %v, got: %v", nextEpoch, indexRoot,
			newState.LatestIndexRootHash32S[nextEpoch],
		)
	}
}

func TestUnslashedAttestingIndices_CanSortAndFilter(t *testing.T) {
	// Generate 2 attestations.
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:  params.BeaconConfig().GenesisSlot + uint64(i),
				Shard: uint64(i + 1),
			},
			AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0,
				0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0},
		}
	}

	// Generate validators and state for the 2 attestations.
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: validators,
	}

	indices, err := UnslashedAttestingIndices(state, atts)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(indices)-1; i++ {
		if indices[i] > indices[i+1] {
			t.Error("sorted indices not sorted")
		}
	}

	// Verify the slashed validator is filtered.
	slashedValidator := indices[0]
	state.ValidatorRegistry[slashedValidator].Slashed = true
	indices, err = UnslashedAttestingIndices(state, atts)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(indices); i++ {
		if indices[i] == slashedValidator {
			t.Errorf("Slashed validator %d is not filtered", slashedValidator)
		}
	}
}

func TestUnslashedAttestingIndices_CantGetIndicesBitfieldError(t *testing.T) {
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot: params.BeaconConfig().GenesisSlot + uint64(i),
			},
			AggregationBitfield: []byte{0xFF},
		}
	}

	state := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
	}
	const wantedErr = "could not get attester indices: wanted participants bitfield length 16, got: 1"
	if _, err := UnslashedAttestingIndices(state, atts); !strings.Contains(err.Error(), wantedErr) {
		t.Errorf("wanted: %v, got: %v", wantedErr, err.Error())
	}
}

func TestAttestingBalance_CorrectBalance(t *testing.T) {
	// Generate 2 attestations.
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:  params.BeaconConfig().GenesisSlot + uint64(i),
				Shard: uint64(i + 1),
			},
			AggregationBitfield: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		}
	}

	// Generate validators with balances and state for the 2 attestations.
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	balances := make([]uint64, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
		balances[i] = params.BeaconConfig().MaxDepositAmount
	}
	state := &pb.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: validators,
		Balances:          balances,
	}

	balance, err := AttestingBalance(state, atts)
	if err != nil {
		t.Fatal(err)
	}
	wanted := 256 * params.BeaconConfig().MaxDepositAmount
	if balance != wanted {
		t.Errorf("wanted balance: %d, got: %d", wanted, balance)
	}
}

func TestAttestingBalance_CantGetIndicesBitfieldError(t *testing.T) {
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot: params.BeaconConfig().GenesisSlot + uint64(i),
			},
			AggregationBitfield: []byte{0xFF},
		}
	}

	state := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
	}
	const wantedErr = "could not get attester indices: wanted participants bitfield length 16, got: 1"
	if _, err := AttestingBalance(state, atts); !strings.Contains(err.Error(), wantedErr) {
		t.Errorf("wanted: %v, got: %v", wantedErr, err.Error())
	}
}

func TestEarliestAttestation_CanGetEarliest(t *testing.T) {
	// Generate 2 attestations.
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:  params.BeaconConfig().GenesisSlot + uint64(i),
				Shard: uint64(i + 1),
			},
			InclusionSlot: uint64(i + 100),
			AggregationBitfield: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		}
	}

	// Generate validators with balances and state for the 2 attestations.
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	balances := make([]uint64, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
		balances[i] = params.BeaconConfig().MaxDepositAmount
	}
	state := &pb.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: validators,
		Balances:          balances,
	}

	// Get attestation for validator index 255.
	idx := uint64(255)
	att, err := EarlistAttestation(state, atts, idx)
	if err != nil {
		t.Fatal(err)
	}
	wantedInclusion := uint64(100)
	if att.InclusionSlot != wantedInclusion {
		t.Errorf("wanted inclusion slot: %d, got: %d", wantedInclusion, att.InclusionSlot)

	}
}

func TestEarliestAttestation_CantGetIndicesBitfieldError(t *testing.T) {
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot: params.BeaconConfig().GenesisSlot + uint64(i),
			},
			AggregationBitfield: []byte{0xFF},
		}
	}

	state := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
	}
	const wantedErr = "could not get attester indices: wanted participants bitfield length 16, got: 1"
	if _, err := EarlistAttestation(state, atts, 0); !strings.Contains(err.Error(), wantedErr) {
		t.Errorf("wanted: %v, got: %v", wantedErr, err.Error())
	}
}

func TestMatchAttestations_PrevEpoch(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	s := params.BeaconConfig().GenesisSlot

	// The correct epoch for source is the first epoch
	// The correct vote for target is '1'
	// The correct vote for head is '2'
	prevAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + 1}},                                                    // source
		{Data: &pb.AttestationData{Slot: s + 1, TargetRoot: []byte{1}}},                             // source, target
		{Data: &pb.AttestationData{Slot: s + 1, TargetRoot: []byte{3}}},                             // source
		{Data: &pb.AttestationData{Slot: s + 1, TargetRoot: []byte{1}}},                             // source, target
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}}},                        // source, head
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{4}}},                        // source
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{1}}}, // source, target, head
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{5}, TargetRoot: []byte{1}}}, // source, target
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{6}}}, // source, head
	}

	currentAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + e + 1}},                                                    // none
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{1}}}, // none
	}

	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}
	state := &pb.BeaconState{
		Slot:                      s + e + 2,
		CurrentEpochAttestations:  currentAtts,
		PreviousEpochAttestations: prevAtts,
		LatestBlockRoots:          blockRoots,
	}

	mAtts, err := MatchAttestations(state, params.BeaconConfig().GenesisEpoch)
	if err != nil {
		t.Fatal(err)
	}

	wantedSrcAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + 1}},
		{Data: &pb.AttestationData{Slot: s + 1, TargetRoot: []byte{1}}},
		{Data: &pb.AttestationData{Slot: s + 1, TargetRoot: []byte{3}}},
		{Data: &pb.AttestationData{Slot: s + 1, TargetRoot: []byte{1}}},
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}}},
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{4}}},
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{1}}},
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{5}, TargetRoot: []byte{1}}},
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{6}}},
	}
	if !reflect.DeepEqual(mAtts.source, wantedSrcAtts) {
		t.Error("source attestations don't match")
	}

	wantedTgtAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + 1, TargetRoot: []byte{1}}},
		{Data: &pb.AttestationData{Slot: s + 1, TargetRoot: []byte{1}}},
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{1}}},
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{5}, TargetRoot: []byte{1}}},
	}
	if !reflect.DeepEqual(mAtts.target, wantedTgtAtts) {
		t.Error("target attestations don't match")
	}

	wantedHeadAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}}},
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{1}}},
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{6}}},
	}
	if !reflect.DeepEqual(mAtts.head, wantedHeadAtts) {
		t.Error("head attestations don't match")
	}
}

func TestMatchAttestations_CurrentEpoch(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	s := params.BeaconConfig().GenesisSlot

	// The correct epoch for source is the first epoch
	// The correct vote for target is '65'
	// The correct vote for head is '66'
	prevAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + 1}},                                                    // none
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{1}}}, // none
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{5}, TargetRoot: []byte{1}}}, // none
		{Data: &pb.AttestationData{Slot: s + 1, BeaconBlockRoot: []byte{2}, TargetRoot: []byte{6}}}, // none
	}

	currentAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + e + 1}},                                                      // source
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{66}, TargetRoot: []byte{65}}}, // source, target, head
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{69}, TargetRoot: []byte{65}}}, // source, target
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{66}, TargetRoot: []byte{68}}}, // source, head
	}

	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}
	state := &pb.BeaconState{
		Slot:                      s + e + 2,
		CurrentEpochAttestations:  currentAtts,
		PreviousEpochAttestations: prevAtts,
		LatestBlockRoots:          blockRoots,
	}

	mAtts, err := MatchAttestations(state, params.BeaconConfig().GenesisEpoch+1)
	if err != nil {
		t.Fatal(err)
	}

	wantedSrcAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + e + 1}},
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{66}, TargetRoot: []byte{65}}},
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{69}, TargetRoot: []byte{65}}},
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{66}, TargetRoot: []byte{68}}},
	}
	if !reflect.DeepEqual(mAtts.source, wantedSrcAtts) {
		t.Error("source attestations don't match")
	}

	wantedTgtAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{66}, TargetRoot: []byte{65}}},
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{69}, TargetRoot: []byte{65}}},
	}
	if !reflect.DeepEqual(mAtts.target, wantedTgtAtts) {
		t.Error("target attestations don't match")
	}

	wantedHeadAtts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{66}, TargetRoot: []byte{65}}},
		{Data: &pb.AttestationData{Slot: s + e + 1, BeaconBlockRoot: []byte{66}, TargetRoot: []byte{68}}},
	}
	if !reflect.DeepEqual(mAtts.head, wantedHeadAtts) {
		t.Error("head attestations don't match")
	}
}

func TestMatchAttestations_EpochOutOfBound(t *testing.T) {
	_, err := MatchAttestations(&pb.BeaconState{Slot: 1}, 2 /* epoch */)
	if !strings.Contains(err.Error(), "input epoch: 2 != current epoch: 0") {
		t.Fatal("Did not receive wanted error")
	}
}

func TestCrosslinkFromAttsData_CanGetCrosslink(t *testing.T) {
	s := &pb.BeaconState{
		CurrentCrosslinks: []*pb.Crosslink{
			{Epoch: params.BeaconConfig().GenesisEpoch},
		},
	}
	slot := (params.BeaconConfig().GenesisEpoch + 100) * params.BeaconConfig().SlotsPerEpoch
	a := &pb.AttestationData{
		Slot:                  slot,
		CrosslinkDataRoot:     []byte{'A'},
		PreviousCrosslinkRoot: []byte{'B'},
	}
	if !proto.Equal(CrosslinkFromAttsData(s, a), &pb.Crosslink{
		Epoch:                       params.BeaconConfig().GenesisEpoch + params.BeaconConfig().MaxCrosslinkEpochs,
		CrosslinkDataRootHash32:     []byte{'A'},
		PreviousCrosslinkRootHash32: []byte{'B'},
	}) {
		t.Error("Incorrect crosslink")
	}
}

func TestAttsForCrosslink_CanGetAttestations(t *testing.T) {
	s := &pb.BeaconState{
		CurrentCrosslinks: []*pb.Crosslink{
			{Epoch: params.BeaconConfig().GenesisEpoch},
		},
	}
	c := &pb.Crosslink{
		CrosslinkDataRootHash32: []byte{'B'},
	}
	atts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{CrosslinkDataRoot: []byte{'A'}}},
		{Data: &pb.AttestationData{CrosslinkDataRoot: []byte{'B'}}}, // Selected
		{Data: &pb.AttestationData{CrosslinkDataRoot: []byte{'C'}}},
		{Data: &pb.AttestationData{CrosslinkDataRoot: []byte{'B'}}}} // Selected
	if !reflect.DeepEqual(attsForCrosslink(s, c, atts), []*pb.PendingAttestation{
		{Data: &pb.AttestationData{CrosslinkDataRoot: []byte{'B'}}},
		{Data: &pb.AttestationData{CrosslinkDataRoot: []byte{'B'}}}}) {
		t.Error("Incorrect attestations for crosslink")
	}
}

func TestCrosslinkAttestingIndices_CanGetIndices(t *testing.T) {
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                  params.BeaconConfig().GenesisSlot + uint64(i),
				Shard:                 uint64(i + 1),
				PreviousCrosslinkRoot: []byte{'E'},
			},
			AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0,
				0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0},
		}
	}

	// Generate validators and state for the 2 attestations.
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	s := &pb.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: validators,
		CurrentCrosslinks: []*pb.Crosslink{
			{Epoch: params.BeaconConfig().GenesisEpoch},
			{Epoch: params.BeaconConfig().GenesisEpoch},
			{Epoch: params.BeaconConfig().GenesisEpoch},
		},
	}
	c := &pb.Crosslink{
		Epoch:                       params.BeaconConfig().GenesisEpoch,
		PreviousCrosslinkRootHash32: []byte{'E'},
	}
	indices, err := CrosslinkAttestingIndices(s, c, atts)
	if err != nil {
		t.Fatal(err)
	}

	// verify the there's indices and it's sorted.
	if len(indices) == 0 {
		t.Error("crosslink attesting indices length can't be 0")
	}
	for i := 0; i < len(indices)-1; i++ {
		if indices[i] > indices[i+1] {
			t.Error("sorted indices not sorted")
		}
	}
}

func TestWinningCrosslink_CantGetMatchingAtts(t *testing.T) {
	wanted := fmt.Sprintf("could not get matching attestations: input epoch: %d != current epoch: %d or previous epoch: %d",
		100, params.BeaconConfig().GenesisEpoch, params.BeaconConfig().GenesisEpoch)
	_, err := WinningCrosslink(&pb.BeaconState{Slot: params.BeaconConfig().GenesisSlot}, 0, 100)
	if err.Error() != wanted {
		t.Fatal(err)
	}
}

func TestWinningCrosslink_ReturnGensisCrosslink(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	gs := params.BeaconConfig().GenesisSlot
	ge := params.BeaconConfig().GenesisEpoch

	state := &pb.BeaconState{
		Slot:                      gs + e + 2,
		PreviousEpochAttestations: []*pb.PendingAttestation{},
		LatestBlockRoots:          make([][]byte, 128),
		CurrentCrosslinks:         []*pb.Crosslink{{Epoch: ge}},
	}

	gCrosslink := &pb.Crosslink{
		Epoch:                       params.BeaconConfig().GenesisEpoch,
		CrosslinkDataRootHash32:     params.BeaconConfig().ZeroHash[:],
		PreviousCrosslinkRootHash32: params.BeaconConfig().ZeroHash[:],
	}

	crosslink, err := WinningCrosslink(state, 0, ge)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(crosslink, gCrosslink) {
		t.Errorf("Did not get genesis crosslink, got: %v", crosslink)
	}
}

func TestWinningCrosslink_CanGetWinningRoot(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	gs := params.BeaconConfig().GenesisSlot
	ge := params.BeaconConfig().GenesisEpoch

	atts := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: gs + 1, CrosslinkDataRoot: []byte{'A'}}},
		{Data: &pb.AttestationData{Slot: gs + 1, CrosslinkDataRoot: []byte{'B'}}}, // winner
		{Data: &pb.AttestationData{Slot: gs + 1, CrosslinkDataRoot: []byte{'C'}}},
	}

	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}

	state := &pb.BeaconState{
		Slot:                      gs + e + 2,
		PreviousEpochAttestations: atts,
		LatestBlockRoots:          blockRoots,
		CurrentCrosslinks:         []*pb.Crosslink{{Epoch: ge, CrosslinkDataRootHash32: []byte{'B'}}},
	}

	winner, err := WinningCrosslink(state, 0, ge)
	if err != nil {
		t.Fatal(err)
	}

	want := &pb.AttestationData{Slot: gs + 1, CrosslinkDataRoot: []byte{'B'}}
	if reflect.DeepEqual(winner, want) {
		t.Errorf("Did not get genesis crosslink, got: %v", winner)
	}
}
