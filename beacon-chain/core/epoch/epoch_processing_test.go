package epoch

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	helpers.ClearShuffledValidatorCache()

	// TODO(2312): remove this and use the mainnet count.
	c := params.BeaconConfig()
	c.MinGenesisActiveValidatorCount = 16384
	params.OverrideBeaconConfig(c)
}

func TestUnslashedAttestingIndices_CanSortAndFilter(t *testing.T) {
	// Generate 2 attestations.
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{},
				Target: &ethpb.Checkpoint{Epoch: 0},
				Crosslink: &ethpb.Crosslink{
					Shard: uint64(i),
				},
			},
			AggregationBits: bitfield.Bitlist{0xFF, 0xFF, 0xFF},
		}
	}

	// Generate validators and state for the 2 attestations.
	validatorCount := 1000
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	indices, err := unslashedAttestingIndices(state, atts)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(indices)-1; i++ {
		if indices[i] >= indices[i+1] {
			t.Error("sorted indices not sorted or duplicated")
		}
	}

	// Verify the slashed validator is filtered.
	slashedValidator := indices[0]
	state.Validators[slashedValidator].Slashed = true
	indices, err = unslashedAttestingIndices(state, atts)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(indices); i++ {
		if indices[i] == slashedValidator {
			t.Errorf("Slashed validator %d is not filtered", slashedValidator)
		}
	}
}

func TestUnslashedAttestingIndices_DuplicatedAttestations(t *testing.T) {
	// Generate 5 of the same attestations.
	atts := make([]*pb.PendingAttestation, 5)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{},
				Target:    &ethpb.Checkpoint{Epoch: 0},
				Crosslink: &ethpb.Crosslink{},
			},
			AggregationBits: bitfield.Bitlist{0xFF, 0xFF, 0xFF},
		}
	}

	// Generate validators and state for the 5 attestations.
	validatorCount := 1000
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	state := &pb.BeaconState{
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	indices, err := unslashedAttestingIndices(state, atts)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(indices)-1; i++ {
		if indices[i] >= indices[i+1] {
			t.Error("sorted indices not sorted or duplicated")
		}
	}
}

func TestAttestingBalance_CorrectBalance(t *testing.T) {
	helpers.ClearAllCaches()

	// Generate 2 attestations.
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard: uint64(i),
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01},
		}
	}

	// Generate validators with balances and state for the 2 attestations.
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	balances := make([]uint64, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	state := &pb.BeaconState{
		Slot:             0,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Validators:       validators,
		Balances:         balances,
	}

	balance, err := AttestingBalance(state, atts)
	if err != nil {
		t.Fatal(err)
	}
	wanted := 256 * params.BeaconConfig().MaxEffectiveBalance
	if balance != wanted {
		t.Errorf("wanted balance: %d, got: %d", wanted, balance)
	}
}

func TestMatchAttestations_PrevEpoch(t *testing.T) {
	helpers.ClearAllCaches()
	e := params.BeaconConfig().SlotsPerEpoch
	s := uint64(0) // slot

	// The correct epoch for source is the first epoch
	// The correct vote for target is '1'
	// The correct vote for head is '2'
	prevAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{}}},                                             // source
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},                              // source, target
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{Root: []byte{3}}}},                              // source
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},                              // source, target
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{}}},                // source, head
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{4}, Target: &ethpb.Checkpoint{}}},                 // source
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{1}}}}, // source, target, head
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{5}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},  // source, target
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{6}}}}, // source, head
	}

	currentAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + e + 1}, Target: &ethpb.Checkpoint{}}},                                            // none
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + e + 1}, BeaconBlockRoot: []byte{2}, Target: &ethpb.Checkpoint{Root: []byte{1}}}}, // none
	}

	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}
	state := &pb.BeaconState{
		Slot:                      s + e + 2,
		CurrentEpochAttestations:  currentAtts,
		PreviousEpochAttestations: prevAtts,
		BlockRoots:                blockRoots,
		RandaoMixes:               make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:          make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	mAtts, err := MatchAttestations(state, 0)
	if err != nil {
		t.Fatal(err)
	}

	wantedSrcAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{Root: []byte{3}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{4}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{5}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{6}}}},
	}
	if !reflect.DeepEqual(mAtts.source, wantedSrcAtts) {
		t.Error("source attestations don't match")
	}

	wantedTgtAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{5}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
	}
	if !reflect.DeepEqual(mAtts.Target, wantedTgtAtts) {
		t.Error("target attestations don't match")
	}

	wantedHeadAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{6}}}},
	}
	if !reflect.DeepEqual(mAtts.head, wantedHeadAtts) {
		t.Error("head attestations don't match")
	}
}

func TestMatchAttestations_CurrentEpoch(t *testing.T) {
	helpers.ClearAllCaches()
	e := params.BeaconConfig().SlotsPerEpoch
	s := uint64(0) // slot

	// The correct epoch for source is the first epoch
	// The correct vote for target is '65'
	// The correct vote for head is '66'
	prevAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{}}},                                            // none
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{2}, Target: &ethpb.Checkpoint{Root: []byte{1}}}}, // none
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{5}, Target: &ethpb.Checkpoint{Root: []byte{1}}}}, // none
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{2}, Target: &ethpb.Checkpoint{Root: []byte{6}}}}, // none
	}

	currentAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{}}},                                              // source
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{65}}}}, // source, target, head
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{69}, Target: &ethpb.Checkpoint{Root: []byte{65}}}}, // source, target
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{68}}}}, // source, head
	}

	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}
	state := &pb.BeaconState{
		Slot:                      s + e + 2,
		CurrentEpochAttestations:  currentAtts,
		PreviousEpochAttestations: prevAtts,
		BlockRoots:                blockRoots,
	}

	mAtts, err := MatchAttestations(state, 1)
	if err != nil {
		t.Fatal(err)
	}

	wantedSrcAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{65}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{69}, Target: &ethpb.Checkpoint{Root: []byte{65}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{68}}}},
	}
	if !reflect.DeepEqual(mAtts.source, wantedSrcAtts) {
		t.Error("source attestations don't match")
	}

	wantedTgtAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{65}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{69}, Target: &ethpb.Checkpoint{Root: []byte{65}}}},
	}
	if !reflect.DeepEqual(mAtts.Target, wantedTgtAtts) {
		t.Error("target attestations don't match")
	}

	wantedHeadAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{65}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{Shard: s + 1}, BeaconBlockRoot: []byte{66}, Target: &ethpb.Checkpoint{Root: []byte{68}}}},
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

func TestAttsForCrosslink_CanGetAttestations(t *testing.T) {
	c := &ethpb.Crosslink{
		DataRoot: []byte{'B'},
	}
	atts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{DataRoot: []byte{'A'}}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{DataRoot: []byte{'B'}}, Target: &ethpb.Checkpoint{}}}, // Selected
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{DataRoot: []byte{'C'}}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{DataRoot: []byte{'B'}}, Target: &ethpb.Checkpoint{}}}} // Selected

	if !reflect.DeepEqual(attsForCrosslink(c, atts), []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{DataRoot: []byte{'B'}}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Crosslink: &ethpb.Crosslink{DataRoot: []byte{'B'}}, Target: &ethpb.Checkpoint{}}}}) {
		t.Error("Incorrect attestations for crosslink")
	}
}

func TestWinningCrosslink_CantGetMatchingAtts(t *testing.T) {
	wanted := fmt.Sprintf("could not get matching attestations: input epoch: %d != current epoch: %d or previous epoch: %d",
		100, 0, 0)
	_, _, err := winningCrosslink(&pb.BeaconState{Slot: 0}, 0, 100)
	if err.Error() != wanted {
		t.Fatal(err)
	}
}

func TestWinningCrosslink_ReturnGenesisCrosslink(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	gs := uint64(0) // genesis slot
	ge := uint64(0) // genesis epoch

	state := &pb.BeaconState{
		Slot:                      gs + e + 2,
		PreviousEpochAttestations: []*pb.PendingAttestation{},
		BlockRoots:                make([][]byte, 128),
		CurrentCrosslinks:         []*ethpb.Crosslink{{StartEpoch: ge}},
	}

	gCrosslink := &ethpb.Crosslink{
		StartEpoch: 0,
		DataRoot:   params.BeaconConfig().ZeroHash[:],
		ParentRoot: params.BeaconConfig().ZeroHash[:],
	}

	crosslink, indices, err := winningCrosslink(state, 0, ge)
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != 0 {
		t.Errorf("genesis crosslink indices is not 0, got: %d", len(indices))
	}
	if !reflect.DeepEqual(crosslink, gCrosslink) {
		t.Errorf("Did not get genesis crosslink, got: %v", crosslink)
	}
}

func TestWinningCrosslink_CanGetWinningRoot(t *testing.T) {
	helpers.ClearAllCaches()
	e := params.BeaconConfig().SlotsPerEpoch
	gs := uint64(0) // genesis slot
	ge := uint64(0) // genesis epoch

	atts := []*pb.PendingAttestation{
		{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    1,
					DataRoot: []byte{'A'},
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
		},
		{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    1,
					DataRoot: []byte{'B'}, // Winner
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
		},
		{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    1,
					DataRoot: []byte{'C'},
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
		},
	}

	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}

	crosslinks := make([]*ethpb.Crosslink, params.BeaconConfig().ShardCount)
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks[i] = &ethpb.Crosslink{
			StartEpoch: ge,
			Shard:      1,
			DataRoot:   []byte{'B'},
		}
	}
	state := &pb.BeaconState{
		Slot:                      gs + e + 2,
		PreviousEpochAttestations: atts,
		BlockRoots:                blockRoots,
		CurrentCrosslinks:         crosslinks,
		RandaoMixes:               make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:          make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	winner, indices, err := winningCrosslink(state, 1, ge)
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != 0 {
		t.Errorf("genesis crosslink indices is not 0, got: %d", len(indices))
	}
	want := &ethpb.Crosslink{StartEpoch: ge, Shard: 1, DataRoot: []byte{'B'}}
	if !reflect.DeepEqual(winner, want) {
		t.Errorf("Did not get wanted crosslink, got: %v, want %v", winner, want)
	}
}

func TestProcessCrosslinks_NoUpdate(t *testing.T) {
	helpers.ClearAllCaches()

	validatorCount := 128
	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}

	var crosslinks []*ethpb.Crosslink
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &ethpb.Crosslink{
			StartEpoch: 0,
			DataRoot:   []byte{'A'},
		})
	}
	state := &pb.BeaconState{
		Slot:              params.BeaconConfig().SlotsPerEpoch + 1,
		Validators:        validators,
		Balances:          balances,
		BlockRoots:        blockRoots,
		RandaoMixes:       make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:  make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentCrosslinks: crosslinks,
	}
	newState, err := ProcessCrosslinks(state)
	if err != nil {
		t.Fatal(err)
	}

	wanted := &ethpb.Crosslink{
		StartEpoch: 0,
		DataRoot:   []byte{'A'},
	}
	// Since there has been no attestation, crosslink stayed the same.
	if !reflect.DeepEqual(wanted, newState.CurrentCrosslinks[0]) {
		t.Errorf("Did not get correct crosslink back")
	}
}

func TestProcessCrosslinks_SuccessfulUpdate(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	gs := uint64(0) // genesis slot
	ge := uint64(0) // genesis epoch

	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/8)
	balances := make([]uint64, params.BeaconConfig().MinGenesisActiveValidatorCount/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}

	crosslinks := make([]*ethpb.Crosslink, params.BeaconConfig().ShardCount)
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks[i] = &ethpb.Crosslink{
			StartEpoch: ge,
			DataRoot:   []byte{'B'},
		}
	}
	var atts []*pb.PendingAttestation
	startShard := uint64(960)
	for s := uint64(0); s < params.BeaconConfig().SlotsPerEpoch; s++ {
		atts = append(atts, &pb.PendingAttestation{
			Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{},
				Crosslink: &ethpb.Crosslink{
					Shard:    startShard + s,
					DataRoot: []byte{'B'},
				},
				Target: &ethpb.Checkpoint{Epoch: 0},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
		})
	}
	state := &pb.BeaconState{
		Slot:                      gs + e + 2,
		Validators:                validators,
		PreviousEpochAttestations: atts,
		Balances:                  balances,
		BlockRoots:                blockRoots,
		CurrentCrosslinks:         crosslinks,
		RandaoMixes:               make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:          make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	newState, err := ProcessCrosslinks(state)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(crosslinks[0], newState.CurrentCrosslinks[0]) {
		t.Errorf("Crosslink is not the same")
	}
}

func TestBaseReward_AccurateRewards(t *testing.T) {
	helpers.ClearAllCaches()

	tests := []struct {
		a uint64
		b uint64
		c uint64
	}{
		{params.BeaconConfig().MinDepositAmount, params.BeaconConfig().MinDepositAmount, 404781},
		{30 * 1e9, 30 * 1e9, 2217026},
		{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance, 2289739},
		{40 * 1e9, params.BeaconConfig().MaxEffectiveBalance, 2289739},
	}
	for _, tt := range tests {
		helpers.ClearAllCaches()
		state := &pb.BeaconState{
			Validators: []*ethpb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: tt.b}},
			Balances: []uint64{tt.a},
		}
		c, err := baseReward(state, 0)
		if err != nil {
			t.Fatal(err)
		}
		if c != tt.c {
			t.Errorf("baseReward(%d) = %d, want = %d",
				tt.a, c, tt.c)
		}
	}
}

func TestProcessJustificationAndFinalization_CantJustifyFinalize(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxEffectiveBalance
	state := &pb.BeaconState{
		JustificationBits: []byte{0x00},
		Slot:              params.BeaconConfig().SlotsPerEpoch * 2,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		Validators: []*ethpb.Validator{{ExitEpoch: e, EffectiveBalance: a}, {ExitEpoch: e, EffectiveBalance: a},
			{ExitEpoch: e, EffectiveBalance: a}, {ExitEpoch: e, EffectiveBalance: a}},
	}
	// Since Attested balances are less than total balances, nothing happened.
	newState, err := ProcessJustificationAndFinalization(state, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, newState) {
		t.Error("Did not get the original state")
	}
}

func TestProcessJustificationAndFinalization_NoBlockRootCurrentEpoch(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxEffectiveBalance
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch * 3,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
		JustificationBits:   []byte{0x03}, // 0b0011
		Validators:          []*ethpb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:            []uint64{a, a, a, a}, // validator total balance should be 128000000000
		BlockRoots:          blockRoots,
	}
	attestedBalance := 4 * e * 3 / 2
	_, err := ProcessJustificationAndFinalization(state, 0, attestedBalance)
	want := "could not get block root for current epoch"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatal("Did not receive correct error")
	}
}

func TestProcessJustificationAndFinalization_ConsecutiveEpochs(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxEffectiveBalance
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch*2 + 1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
		JustificationBits:   bitfield.Bitvector4{0x0F}, // 0b1111
		Validators:          []*ethpb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:            []uint64{a, a, a, a}, // validator total balance should be 128000000000
		BlockRoots:          blockRoots,
	}
	attestedBalance := 4 * e * 3 / 2
	newState, err := ProcessJustificationAndFinalization(state, 0, attestedBalance)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newState.CurrentJustifiedCheckpoint.Root, []byte{byte(128)}) {
		t.Errorf("Wanted current justified root: %v, got: %v",
			[]byte{byte(128)}, newState.CurrentJustifiedCheckpoint.Root)
	}
	if newState.CurrentJustifiedCheckpoint.Epoch != 2 {
		t.Errorf("Wanted justified epoch: %d, got: %d",
			2, newState.CurrentJustifiedCheckpoint.Epoch)
	}
	if newState.PreviousJustifiedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted previous justified epoch: %d, got: %d",
			0, newState.PreviousJustifiedCheckpoint.Epoch)
	}
	if !bytes.Equal(newState.FinalizedCheckpoint.Root, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf("Wanted current finalized root: %v, got: %v",
			params.BeaconConfig().ZeroHash, newState.FinalizedCheckpoint.Root)
	}
	if newState.FinalizedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted finalized epoch: 0, got: %d", newState.FinalizedCheckpoint.Epoch)
	}
}

func TestProcessJustificationAndFinalization_JustifyCurrentEpoch(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxEffectiveBalance
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch*2 + 1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
		JustificationBits:   bitfield.Bitvector4{0x03}, // 0b0011
		Validators:          []*ethpb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:            []uint64{a, a, a, a}, // validator total balance should be 128000000000
		BlockRoots:          blockRoots,
	}
	attestedBalance := 4 * e * 3 / 2
	newState, err := ProcessJustificationAndFinalization(state, 0, attestedBalance)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newState.CurrentJustifiedCheckpoint.Root, []byte{byte(128)}) {
		t.Errorf("Wanted current justified root: %v, got: %v",
			[]byte{byte(128)}, newState.CurrentJustifiedCheckpoint.Root)
	}
	if newState.CurrentJustifiedCheckpoint.Epoch != 2 {
		t.Errorf("Wanted justified epoch: %d, got: %d",
			2, newState.CurrentJustifiedCheckpoint.Epoch)
	}
	if newState.PreviousJustifiedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted previous justified epoch: %d, got: %d",
			0, newState.PreviousJustifiedCheckpoint.Epoch)
	}
	if !bytes.Equal(newState.FinalizedCheckpoint.Root, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf("Wanted current finalized root: %v, got: %v",
			params.BeaconConfig().ZeroHash, newState.FinalizedCheckpoint.Root)
	}
	if newState.FinalizedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted finalized epoch: 0, got: %d", newState.FinalizedCheckpoint.Epoch)
	}
}

func TestProcessJustificationAndFinalization_JustifyPrevEpoch(t *testing.T) {
	helpers.ClearAllCaches()
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxEffectiveBalance
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch*2 + 1,
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		JustificationBits: bitfield.Bitvector4{0x03}, // 0b0011
		Validators:        []*ethpb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:          []uint64{a, a, a, a}, // validator total balance should be 128000000000
		BlockRoots:        blockRoots, FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	attestedBalance := 4 * e * 3 / 2
	newState, err := ProcessJustificationAndFinalization(state, attestedBalance, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newState.CurrentJustifiedCheckpoint.Root, []byte{byte(128)}) {
		t.Errorf("Wanted current justified root: %v, got: %v",
			[]byte{byte(128)}, newState.CurrentJustifiedCheckpoint.Root)
	}
	if newState.PreviousJustifiedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted previous justified epoch: %d, got: %d",
			0, newState.PreviousJustifiedCheckpoint.Epoch)
	}
	if newState.CurrentJustifiedCheckpoint.Epoch != 2 {
		t.Errorf("Wanted justified epoch: %d, got: %d",
			2, newState.CurrentJustifiedCheckpoint.Epoch)
	}
	if !bytes.Equal(newState.FinalizedCheckpoint.Root, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf("Wanted current finalized root: %v, got: %v",
			params.BeaconConfig().ZeroHash, newState.FinalizedCheckpoint.Root)
	}
	if newState.FinalizedCheckpoint.Epoch != 0 {
		t.Errorf("Wanted finalized epoch: 0, got: %d", newState.FinalizedCheckpoint.Epoch)
	}
}

func TestProcessSlashings_NotSlashed(t *testing.T) {
	s := &pb.BeaconState{
		Slot:       0,
		Validators: []*ethpb.Validator{{Slashed: true}},
		Balances:   []uint64{params.BeaconConfig().MaxEffectiveBalance},
		Slashings:  []uint64{0, 1e9},
	}
	newState, err := ProcessSlashings(s)
	if err != nil {
		t.Fatal(err)
	}
	wanted := params.BeaconConfig().MaxEffectiveBalance
	if newState.Balances[0] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, newState.Balances[0])
	}
}

func TestProcessSlashings_SlashedLess(t *testing.T) {

	tests := []struct {
		state *pb.BeaconState
		want  uint64
	}{
		{
			state: &pb.BeaconState{
				Validators: []*ethpb.Validator{
					{Slashed: true,
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector / 2,
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance}},
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
				Slashings: []uint64{0, 1e9},
			},
			// penalty    = validator balance / increment * (3*total_penalties) / total_balance * increment
			// 3000000000 = (32 * 1e9)        / (1 * 1e9) * (3*1e9)             / (32*1e9)      * (1 * 1e9)
			want: uint64(29000000000), // 32 * 1e9 - 3000000000
		},
		{
			state: &pb.BeaconState{
				Validators: []*ethpb.Validator{
					{Slashed: true,
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector / 2,
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
				Slashings: []uint64{0, 1e9},
			},
			// penalty    = validator balance / increment * (3*total_penalties) / total_balance * increment
			// 1000000000 = (32 * 1e9)        / (1 * 1e9) * (3*1e9)             / (64*1e9)      * (1 * 1e9)
			want: uint64(31000000000), // 32 * 1e9 - 1000000000
		},
		{
			state: &pb.BeaconState{
				Validators: []*ethpb.Validator{
					{Slashed: true,
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector / 2,
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
				Slashings: []uint64{0, 2 * 1e9},
			},
			// penalty    = validator balance / increment * (3*total_penalties) / total_balance * increment
			// 3000000000 = (32 * 1e9)        / (1 * 1e9) * (3*2e9)             / (64*1e9)      * (1 * 1e9)
			want: uint64(29000000000), // 32 * 1e9 - 3000000000
		},
		{
			state: &pb.BeaconState{
				Validators: []*ethpb.Validator{
					{Slashed: true,
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector / 2,
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement}},
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement, params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement},
				Slashings: []uint64{0, 1e9},
			},
			// penalty    = validator balance           / increment * (3*total_penalties) / total_balance        * increment
			// 3000000000 = (32  * 1e9 - 1*1e9)         / (1 * 1e9) * (3*1e9)             / (31*1e9)             * (1 * 1e9)
			want: uint64(28000000000), // 31 * 1e9 - 3000000000
		},
	}

	for i, tt := range tests {
		t.Run(string(i), func(t *testing.T) {
			helpers.ClearAllCaches()

			original := proto.Clone(tt.state)
			newState, err := ProcessSlashings(tt.state)
			if err != nil {
				t.Fatal(err)
			}

			if newState.Balances[0] != tt.want {
				t.Errorf(
					"ProcessSlashings({%v}) = newState; newState.Balances[0] = %d; wanted %d",
					original,
					newState.Balances[0],
					tt.want,
				)
			}
		})
	}
}

func TestProcessFinalUpdates_CanProcess(t *testing.T) {
	s := buildState(params.BeaconConfig().SlotsPerHistoricalRoot-1, params.BeaconConfig().SlotsPerEpoch)
	ce := helpers.CurrentEpoch(s)
	ne := ce + 1
	s.Eth1DataVotes = []*ethpb.Eth1Data{}
	s.Balances[0] = 29 * 1e9
	s.Slashings[ce] = 0
	s.RandaoMixes[ce] = []byte{'A'}
	newS, err := ProcessFinalUpdates(s)
	if err != nil {
		t.Fatal(err)
	}

	// Verify effective balance is correctly updated.
	if newS.Validators[0].EffectiveBalance != 29*1e9 {
		t.Errorf("effective balance incorrectly updated, got %d", s.Validators[0].EffectiveBalance)
	}

	// Verify start shard is correctly updated.
	if newS.StartShard != 64 {
		t.Errorf("start shard incorrectly updated, got %d", 64)
	}

	// Verify latest active index root is correctly updated in the right position.
	pos := (ne + params.BeaconConfig().ActivationExitDelay) % params.BeaconConfig().EpochsPerHistoricalVector
	if bytes.Equal(newS.ActiveIndexRoots[pos], params.BeaconConfig().ZeroHash[:]) {
		t.Error("latest active index roots still zero hashes")
	}

	// Verify slashed balances correctly updated.
	if newS.Slashings[ce] != newS.Slashings[ne] {
		t.Errorf("wanted slashed balance %d, got %d",
			newS.Slashings[ce],
			newS.Slashings[ne])
	}

	// Verify randao is correctly updated in the right position.
	if bytes.Equal(newS.RandaoMixes[ne], params.BeaconConfig().ZeroHash[:]) {
		t.Error("latest RANDAO still zero hashes")
	}

	// Verify historical root accumulator was appended.
	if len(newS.HistoricalRoots) != 1 {
		t.Errorf("wanted slashed balance %d, got %d", 1, len(newS.HistoricalRoots[ce]))
	}

	if newS.CurrentEpochAttestations == nil {
		t.Error("nil value stored in current epoch attestations instead of empty slice")
	}
}

func TestCrosslinkDelta_NoOneAttested(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	validatorCount := uint64(128)
	state := buildState(e+2, validatorCount)

	rewards, penalties, err := crosslinkDelta(state)
	if err != nil {
		t.Fatal(err)
	}
	for i := uint64(0); i < validatorCount; i++ {
		// Since no one attested, all the validators should gain 0 reward
		if rewards[i] != 0 {
			t.Errorf("Wanted reward balance 0, got %d", rewards[i])
		}
		// Since no one attested, all the validators should get penalized the same
		base, err := baseReward(state, i)
		if err != nil {
			t.Fatal(err)
		}
		if penalties[i] != base {
			t.Errorf("Wanted penalty balance %d, got %d",
				base, penalties[i])
		}
	}
}

func TestProcessRegistryUpdates_NoRotation(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
		},
		Balances: []uint64{
			params.BeaconConfig().MaxEffectiveBalance,
			params.BeaconConfig().MaxEffectiveBalance,
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	newState, err := ProcessRegistryUpdates(state)
	if err != nil {
		t.Fatal(err)
	}
	for i, validator := range newState.Validators {
		if validator.ExitEpoch != params.BeaconConfig().ActivationExitDelay {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().ActivationExitDelay, validator.ExitEpoch)
		}
	}
}

func TestCrosslinkDelta_SomeAttested(t *testing.T) {
	helpers.ClearAllCaches()
	e := params.BeaconConfig().SlotsPerEpoch
	helpers.ClearShuffledValidatorCache()
	state := buildState(e+2, params.BeaconConfig().MinGenesisActiveValidatorCount/8)
	startShard := uint64(960)
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    startShard + uint64(i),
					DataRoot: []byte{'A'},
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			InclusionDelay:  uint64(i + 100),
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
		}
	}
	state.PreviousEpochAttestations = atts
	state.CurrentCrosslinks[startShard] = &ethpb.Crosslink{
		DataRoot: []byte{'A'}, Shard: startShard,
	}
	state.CurrentCrosslinks[startShard+1] = &ethpb.Crosslink{
		DataRoot: []byte{'A'}, Shard: startShard + 1,
	}

	rewards, penalties, err := crosslinkDelta(state)
	if err != nil {
		t.Fatal(err)
	}

	attestedIndices := []uint64{5, 16, 336, 797, 1082, 1450, 1770, 1958}
	for _, i := range attestedIndices {
		// Since all these validators attested, they should get the same rewards.
		want := uint64(12649)
		if rewards[i] != want {
			t.Errorf("Wanted reward balance %d, got %d", want, rewards[i])
		}
		// Since all these validators attested, they shouldn't get penalized.
		if penalties[i] != 0 {
			t.Errorf("Wanted penalty balance 0, got %d", penalties[i])
		}
	}

	nonAttestedIndices := []uint64{12, 23, 45, 79}
	for _, i := range nonAttestedIndices {
		base, err := baseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}
		wanted := base
		// Since all these validators did not attest, they shouldn't get rewarded.
		if rewards[i] != 0 {
			t.Errorf("Wanted reward balance 0, got %d", rewards[i])
		}
		// Base penalties for not attesting.
		if penalties[i] != wanted {
			t.Errorf("Wanted penalty balance %d, got %d", wanted, penalties[i])
		}
	}
}

func TestCrosslinkDelta_CantGetWinningCrosslink(t *testing.T) {
	state := buildState(0, 1)

	_, _, err := crosslinkDelta(state)
	wanted := "could not get winning crosslink: could not get matching attestations"
	if !strings.Contains(err.Error(), wanted) {
		t.Fatalf("Got: %v, want: %v", err.Error(), wanted)
	}
}

func TestAttestationDelta_CantGetBlockRoot(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	state := buildState(2*e, 1)
	state.Slot = 0

	_, _, err := attestationDelta(state)
	wanted := "could not get block root for epoch"
	if !strings.Contains(err.Error(), wanted) {
		t.Fatalf("Got: %v, want: %v", err.Error(), wanted)
	}
}

func TestAttestationDelta_CantGetAttestation(t *testing.T) {
	state := buildState(0, 1)

	_, _, err := attestationDelta(state)
	wanted := "could not get source, target and head attestations"
	if !strings.Contains(err.Error(), wanted) {
		t.Fatalf("Got: %v, want: %v", err.Error(), wanted)
	}
}

func TestAttestationDelta_CantGetAttestationIndices(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	state := buildState(e+2, 1)
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard: uint64(i),
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			InclusionDelay:  uint64(i + 100),
			AggregationBits: bitfield.Bitlist{0xFF, 0x01},
		}
	}
	state.PreviousEpochAttestations = atts

	_, _, err := attestationDelta(state)
	wanted := "could not get attestation indices"
	if !strings.Contains(err.Error(), wanted) {
		t.Fatalf("Got: %v, want: %v", err.Error(), wanted)
	}
}

func TestAttestationDelta_NoOneAttested(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount / 32
	state := buildState(e+2, validatorCount)
	//startShard := uint64(960)
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    uint64(i),
					DataRoot: []byte{'A'},
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			InclusionDelay:  uint64(i + 100),
			AggregationBits: bitfield.Bitlist{0xC0, 0x01},
		}
	}

	rewards, penalties, err := attestationDelta(state)
	if err != nil {
		t.Fatal(err)
	}
	for i := uint64(0); i < validatorCount; i++ {
		// Since no one attested, all the validators should gain 0 reward
		if rewards[i] != 0 {
			t.Errorf("Wanted reward balance 0, got %d", rewards[i])
		}
		// Since no one attested, all the validators should get penalized the same
		// it's 3 times the penalized amount because source, target and head.
		base, err := baseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}
		wanted := 3 * base
		if penalties[i] != wanted {
			t.Errorf("Wanted penalty balance %d, got %d",
				wanted, penalties[i])
		}
	}
}

func TestAttestationDelta_SomeAttested(t *testing.T) {
	helpers.ClearAllCaches()
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount / 8
	state := buildState(e+2, validatorCount)
	startShard := uint64(960)
	atts := make([]*pb.PendingAttestation, 3)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    startShard + uint64(i),
					DataRoot: []byte{'A'},
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	state.PreviousEpochAttestations = atts
	state.CurrentCrosslinks[startShard] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}
	state.CurrentCrosslinks[startShard+1] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}

	rewards, penalties, err := attestationDelta(state)
	if err != nil {
		t.Fatal(err)
	}

	attestedBalance, err := AttestingBalance(state, atts)
	if err != nil {
		t.Error(err)
	}
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		t.Fatal(err)
	}

	attestedIndices := []uint64{5, 754, 797, 1637, 1770, 1862, 1192}
	for _, i := range attestedIndices {
		base, err := baseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}
		// Base rewards for getting source right
		wanted := 3 * (base * attestedBalance / totalBalance)
		// Base rewards for proposer and attesters working together getting attestation
		// on chain in the fatest manner
		proposerReward := base / params.BeaconConfig().ProposerRewardQuotient
		wanted += (base - proposerReward) * params.BeaconConfig().MinAttestationInclusionDelay
		if rewards[i] != wanted {
			t.Errorf("Wanted reward balance %d, got %d", wanted, rewards[i])
		}
		// Since all these validators attested, they shouldn't get penalized.
		if penalties[i] != 0 {
			t.Errorf("Wanted penalty balance 0, got %d", penalties[i])
		}
	}

	nonAttestedIndices := []uint64{12, 23, 45, 79}
	for _, i := range nonAttestedIndices {
		base, err := baseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}
		wanted := 3 * base
		// Since all these validators did not attest, they shouldn't get rewarded.
		if rewards[i] != 0 {
			t.Errorf("Wanted reward balance 0, got %d", rewards[i])
		}
		// Base penalties for not attesting.
		if penalties[i] != wanted {
			t.Errorf("Wanted penalty balance %d, got %d", wanted, penalties[i])
		}
	}
}

func TestAttestationDelta_SomeAttestedFinalityDelay(t *testing.T) {
	helpers.ClearAllCaches()
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount / 8
	state := buildState(e+4, validatorCount)
	startShard := uint64(960)
	atts := make([]*pb.PendingAttestation, 3)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    startShard + uint64(i),
					DataRoot: []byte{'A'},
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	state.PreviousEpochAttestations = atts
	state.FinalizedCheckpoint.Epoch = 0
	state.CurrentCrosslinks[startShard] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}
	state.CurrentCrosslinks[startShard+1] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}

	rewards, penalties, err := attestationDelta(state)
	if err != nil {
		t.Fatal(err)
	}

	attestedBalance, err := AttestingBalance(state, atts)
	if err != nil {
		t.Error(err)
	}
	totalBalance, err := helpers.TotalActiveBalance(state)
	if err != nil {
		t.Fatal(err)
	}

	attestedIndices := []uint64{5, 754, 797, 1637, 1770, 1862, 1192}
	for _, i := range attestedIndices {
		base, err := baseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}
		// Base rewards for getting source right
		wanted := 3 * (base * attestedBalance / totalBalance)
		// Base rewards for proposer and attesters working together getting attestation
		// on chain in the fatest manner
		proposerReward := base / params.BeaconConfig().ProposerRewardQuotient
		wanted += (base - proposerReward) * params.BeaconConfig().MinAttestationInclusionDelay
		if rewards[i] != wanted {
			t.Errorf("Wanted reward balance %d, got %d", wanted, rewards[i])
		}
		// Since all these validators attested, they shouldn't get penalized.
		if penalties[i] != 0 {
			t.Errorf("Wanted penalty balance 0, got %d", penalties[i])
		}
	}

	nonAttestedIndices := []uint64{12, 23, 45, 79}
	for _, i := range nonAttestedIndices {
		base, err := baseReward(state, i)
		if err != nil {
			t.Errorf("Could not get base reward: %v", err)
		}
		wanted := 3 * base
		// Since all these validators did not attest, they shouldn't get rewarded.
		if rewards[i] != 0 {
			t.Errorf("Wanted reward balance 0, got %d", rewards[i])
		}
		// Base penalties for not attesting.
		if penalties[i] != wanted {
			t.Errorf("Wanted penalty balance %d, got %d", wanted, penalties[i])
		}
	}
}

func TestProcessRegistryUpdates_EligibleToActivate(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                5 * params.BeaconConfig().SlotsPerEpoch,
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	limit, err := helpers.ValidatorChurnLimit(state)
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < int(limit)+10; i++ {
		state.Validators = append(state.Validators, &ethpb.Validator{
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			ActivationEpoch:            params.BeaconConfig().FarFutureEpoch,
		})
	}
	currentEpoch := helpers.CurrentEpoch(state)
	newState, err := ProcessRegistryUpdates(state)
	if err != nil {
		t.Error(err)
	}
	for i, validator := range newState.Validators {
		if validator.ActivationEligibilityEpoch != currentEpoch {
			t.Errorf("Could not update registry %d, wanted activation eligibility epoch %d got %d",
				i, currentEpoch, validator.ActivationEligibilityEpoch)
		}
		if i < int(limit) && validator.ActivationEpoch != helpers.DelayedActivationExitEpoch(currentEpoch) {
			t.Errorf("Could not update registry %d, validators failed to activate: wanted activation epoch %d, got %d",
				i, helpers.DelayedActivationExitEpoch(currentEpoch), validator.ActivationEpoch)
		}
		if i >= int(limit) && validator.ActivationEpoch != params.BeaconConfig().FarFutureEpoch {
			t.Errorf("Could not update registry %d, validators should not have been activated, wanted activation epoch: %d, got %d",
				i, params.BeaconConfig().FarFutureEpoch, validator.ActivationEpoch)
		}
	}
}

func TestProcessRegistryUpdates_ActivationCompletes(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay,
				ActivationEpoch: 5 + params.BeaconConfig().ActivationExitDelay + 1},
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay,
				ActivationEpoch: 5 + params.BeaconConfig().ActivationExitDelay + 1},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	newState, err := ProcessRegistryUpdates(state)
	if err != nil {
		t.Error(err)
	}
	for i, validator := range newState.Validators {
		if validator.ExitEpoch != params.BeaconConfig().ActivationExitDelay {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().ActivationExitDelay, validator.ExitEpoch)
		}
	}
}

func TestProcessRegistryUpdates_ValidatorsEjected(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 0,
		Validators: []*ethpb.Validator{
			{
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: params.BeaconConfig().EjectionBalance - 1,
			},
			{
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: params.BeaconConfig().EjectionBalance - 1,
			},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	newState, err := ProcessRegistryUpdates(state)
	if err != nil {
		t.Error(err)
	}
	for i, validator := range newState.Validators {
		if validator.ExitEpoch != params.BeaconConfig().ActivationExitDelay+1 {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().ActivationExitDelay+1, validator.ExitEpoch)
		}
	}
}

func TestProcessRegistryUpdates_CanExits(t *testing.T) {
	epoch := uint64(5)
	exitEpoch := helpers.DelayedActivationExitEpoch(epoch)
	minWithdrawalDelay := params.BeaconConfig().MinValidatorWithdrawabilityDelay
	state := &pb.BeaconState{
		Slot: epoch * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{
				ExitEpoch:         exitEpoch,
				WithdrawableEpoch: exitEpoch + minWithdrawalDelay},
			{
				ExitEpoch:         exitEpoch,
				WithdrawableEpoch: exitEpoch + minWithdrawalDelay},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	newState, err := ProcessRegistryUpdates(state)
	if err != nil {
		t.Fatal(err)
	}
	for i, validator := range newState.Validators {
		if validator.ExitEpoch != exitEpoch {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i,
				exitEpoch,
				validator.ExitEpoch,
			)
		}
	}
}

func TestProcessRewardsAndPenalties_GenesisEpoch(t *testing.T) {
	state := &pb.BeaconState{Slot: params.BeaconConfig().SlotsPerEpoch - 1, StartShard: 999}
	newState, err := ProcessRewardsAndPenalties(state)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, newState) {
		t.Error("genesis state mutated")
	}
}

func TestProcessRewardsAndPenalties_SomeAttested(t *testing.T) {
	helpers.ClearAllCaches()
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount / 8
	state := buildState(e+2, validatorCount)
	startShard := uint64(960)
	atts := make([]*pb.PendingAttestation, 3)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    startShard + uint64(i),
					DataRoot: []byte{'A'},
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	state.PreviousEpochAttestations = atts
	state.CurrentCrosslinks[startShard] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}
	state.CurrentCrosslinks[startShard+1] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}
	state.CurrentCrosslinks[startShard+2] = &ethpb.Crosslink{
		DataRoot: []byte{'A'},
	}

	state, err := ProcessRewardsAndPenalties(state)
	if err != nil {
		t.Fatal(err)
	}
	wanted := uint64(31999949392)
	if state.Balances[0] != wanted {
		t.Errorf("wanted balance: %d, got: %d",
			wanted, state.Balances[0])
	}
	wanted = uint64(31999995452)
	if state.Balances[4] != wanted {
		t.Errorf("wanted balance: %d, got: %d",
			wanted, state.Balances[1])
	}
}

func buildState(slot uint64, validatorCount uint64) *pb.BeaconState {
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = params.BeaconConfig().ZeroHash[:]
	}
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}
	return &pb.BeaconState{
		Slot:                        slot,
		Balances:                    validatorBalances,
		Validators:                  validators,
		CurrentCrosslinks:           make([]*ethpb.Crosslink, params.BeaconConfig().ShardCount),
		RandaoMixes:                 make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:            make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CompactCommitteesRoots:      make([][]byte, params.BeaconConfig().EpochsPerSlashingsVector),
		Slashings:                   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		BlockRoots:                  make([][]byte, params.BeaconConfig().SlotsPerEpoch*10),
		FinalizedCheckpoint:         &ethpb.Checkpoint{},
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{},
	}
}
