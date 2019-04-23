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
		ValidatorBalances: []uint64{
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
	validatorIndices := helpers.ActiveValidatorIndices(state.ValidatorRegistry, nextEpoch)
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
