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
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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

func TestUnslashedAttestingIndices_CanSortAndFilter(t *testing.T) {
	// Generate 2 attestations.
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:        uint64(i),
				TargetEpoch: 0,
				Shard:       uint64(i + 2),
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
		Slot:                   0,
		ValidatorRegistry:      validators,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
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
				Slot:        uint64(i),
				TargetEpoch: 0,
				Shard:       2,
			},
			AggregationBitfield: []byte{0xff},
		}
	}

	state := &pb.BeaconState{
		Slot:                   0,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	const wantedErr = "could not get attester indices: wanted participants bitfield length 0, got: 1"
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
				Slot:        uint64(i),
				TargetEpoch: 0,
				Shard:       uint64(i + 2),
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
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxDepositAmount,
		}
		balances[i] = params.BeaconConfig().MaxDepositAmount
	}
	state := &pb.BeaconState{
		Slot:                   0,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
		ValidatorRegistry:      validators,
		Balances:               balances,
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
				Slot:        uint64(i),
				TargetEpoch: 0,
				Shard:       2,
			},
			AggregationBitfield: []byte{0xFF},
		}
	}

	state := &pb.BeaconState{
		Slot:                   0,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	const wantedErr = "could not get attester indices: wanted participants bitfield length 0, got: 1"
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
				Slot:        uint64(i),
				TargetEpoch: 0,
				Shard:       uint64(i + 2),
			},
			InclusionDelay: uint64(i + 100),
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
		Slot:                   0,
		ValidatorRegistry:      validators,
		Balances:               balances,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	// Get attestation for validator index 255.
	idx := uint64(914)
	att, err := earlistAttestation(state, atts, idx)
	if err != nil {
		t.Fatal(err)
	}
	wantedInclusion := uint64(18446744073709551615)
	if att.InclusionDelay != wantedInclusion {
		t.Errorf("wanted inclusion slot: %d, got: %d", wantedInclusion, att.InclusionDelay)

	}
}

func TestEarliestAttestation_CantGetIndicesBitfieldError(t *testing.T) {
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:        uint64(i),
				TargetEpoch: 0,
				Shard:       2,
			},
			AggregationBitfield: []byte{0xFF},
		}
	}

	state := &pb.BeaconState{
		Slot:                   0,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	const wantedErr = "could not get attester indices: wanted participants bitfield length 0, got: 1"
	if _, err := earlistAttestation(state, atts, 0); !strings.Contains(err.Error(), wantedErr) {
		t.Errorf("wanted: %v, got: %v", wantedErr, err.Error())
	}
}

func TestMatchAttestations_PrevEpoch(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	s := uint64(0) // slot

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
		LatestRandaoMixes:         make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots:    make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	mAtts, err := MatchAttestations(state, 0)
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
	s := uint64(0) // slot

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

	mAtts, err := MatchAttestations(state, 1)
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
			{Epoch: 0},
		},
	}
	slot := (100) * params.BeaconConfig().SlotsPerEpoch
	a := &pb.AttestationData{
		Slot:                  slot,
		CrosslinkDataRoot:     []byte{'A'},
		PreviousCrosslinkRoot: []byte{'B'},
	}
	if !proto.Equal(CrosslinkFromAttsData(s, a), &pb.Crosslink{
		Epoch:                       params.BeaconConfig().MaxCrosslinkEpochs,
		CrosslinkDataRootHash32:     []byte{'A'},
		PreviousCrosslinkRootHash32: []byte{'B'},
	}) {
		t.Error("Incorrect crosslink")
	}
}

func TestAttsForCrosslink_CanGetAttestations(t *testing.T) {
	s := &pb.BeaconState{
		CurrentCrosslinks: []*pb.Crosslink{
			{Epoch: 0},
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
				Slot:                  uint64(i),
				Shard:                 uint64(i + 2),
				PreviousCrosslinkRoot: []byte{'E'},
				TargetEpoch:           0,
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
		Slot:              0,
		ValidatorRegistry: validators,
		CurrentCrosslinks: []*pb.Crosslink{
			{Epoch: 0},
			{Epoch: 0},
			{Epoch: 0},
			{Epoch: 0},
		},
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	c := &pb.Crosslink{
		Epoch:                       0,
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
		100, 0, 0)
	_, _, err := WinningCrosslink(&pb.BeaconState{Slot: 0}, 0, 100)
	if err.Error() != wanted {
		t.Fatal(err)
	}
}

func TestWinningCrosslink_ReturnGensisCrosslink(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	gs := uint64(0) // genesis slot
	ge := uint64(0) // genesis epoch

	state := &pb.BeaconState{
		Slot:                      gs + e + 2,
		PreviousEpochAttestations: []*pb.PendingAttestation{},
		LatestBlockRoots:          make([][]byte, 128),
		CurrentCrosslinks:         []*pb.Crosslink{{Epoch: ge}},
	}

	gCrosslink := &pb.Crosslink{
		Epoch:                       0,
		CrosslinkDataRootHash32:     params.BeaconConfig().ZeroHash[:],
		PreviousCrosslinkRootHash32: params.BeaconConfig().ZeroHash[:],
	}

	crosslink, indices, err := WinningCrosslink(state, 0, ge)
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != 0 {
		t.Errorf("gensis crosslink indices is not 0, got: %d", len(indices))
	}
	if !reflect.DeepEqual(crosslink, gCrosslink) {
		t.Errorf("Did not get genesis crosslink, got: %v", crosslink)
	}
}

func TestWinningCrosslink_CanGetWinningRoot(t *testing.T) {
	t.Skip()
	// TODO(#2307) unskip after ProcessCrosslinks is finished
	e := params.BeaconConfig().SlotsPerEpoch
	gs := uint64(0) // genesis slot
	ge := uint64(0) // genesis epoch

	atts := []*pb.PendingAttestation{
		{
			Data: &pb.AttestationData{
				Slot:              gs + 1,
				CrosslinkDataRoot: []byte{'A'},
			},
		},
		{
			Data: &pb.AttestationData{
				Slot:              gs + 1,
				CrosslinkDataRoot: []byte{'B'}, // winner
			},
		},
		{
			Data: &pb.AttestationData{
				Slot:              gs + 1,
				CrosslinkDataRoot: []byte{'C'},
			},
		},
	}

	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}

	currentCrosslinks := make([]*pb.Crosslink, params.BeaconConfig().ShardCount)
	currentCrosslinks[3] = &pb.Crosslink{Epoch: ge, CrosslinkDataRootHash32: []byte{'B'}}
	state := &pb.BeaconState{
		Slot:                      gs + e + 2,
		PreviousEpochAttestations: atts,
		LatestBlockRoots:          blockRoots,
		CurrentCrosslinks:         currentCrosslinks,
		LatestRandaoMixes:         make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots:    make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}

	winner, indices, err := WinningCrosslink(state, 0, ge)
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != 0 {
		t.Errorf("gensis crosslink indices is not 0, got: %d", len(indices))
	}
	want := &pb.Crosslink{Epoch: ge, CrosslinkDataRootHash32: []byte{'B'}}
	if !reflect.DeepEqual(winner, want) {
		t.Errorf("Did not get genesis crosslink, got: %v", winner)
	}
}

func TestProcessCrosslink_NoUpdate(t *testing.T) {
	validatorCount := 128
	validators := make([]*pb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxDepositAmount,
		}
		balances[i] = params.BeaconConfig().MaxDepositAmount
	}
	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}
	oldCrosslink := &pb.Crosslink{
		Epoch:                   0,
		CrosslinkDataRootHash32: []byte{'A'},
	}
	var crosslinks []*pb.Crosslink
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &pb.Crosslink{
			Epoch:                   0,
			CrosslinkDataRootHash32: []byte{'A'},
		})
	}
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch + 1,
		ValidatorRegistry:      validators,
		Balances:               balances,
		LatestBlockRoots:       blockRoots,
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
		CurrentCrosslinks:      crosslinks,
	}
	newState, err := ProcessCrosslink(state)
	if err != nil {
		t.Fatal(err)
	}

	// Since there has been no attestation, crosslink stayed the same.
	if !reflect.DeepEqual(oldCrosslink, newState.CurrentCrosslinks[0]) {
		t.Errorf("Did not get correct crosslink back")
	}
}

func TestProcessCrosslink_SuccessfulUpdate(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	gs := uint64(0) // genesis slot
	ge := uint64(0) // genesis epoch

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart/8)
	balances := make([]uint64, params.BeaconConfig().DepositsForChainStart/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxDepositAmount,
		}
		balances[i] = params.BeaconConfig().MaxDepositAmount
	}
	blockRoots := make([][]byte, 128)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i + 1)}
	}

	crosslinks := make([]*pb.Crosslink, params.BeaconConfig().ShardCount)
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks[i] = &pb.Crosslink{
			Epoch:                   ge,
			CrosslinkDataRootHash32: []byte{'B'},
		}
	}
	var atts []*pb.PendingAttestation
	for s := uint64(0); s < params.BeaconConfig().ShardCount; s++ {
		atts = append(atts, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:              gs + 1 + (s % e),
				Shard:             s,
				CrosslinkDataRoot: []byte{'B'},
				TargetEpoch:       0,
			},
			AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0},
		})
	}
	state := &pb.BeaconState{
		Slot:                      gs + e + 2,
		ValidatorRegistry:         validators,
		PreviousEpochAttestations: atts,
		Balances:                  balances,
		LatestBlockRoots:          blockRoots,
		CurrentCrosslinks:         crosslinks,
		LatestRandaoMixes:         make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots:    make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
	}
	newState, err := ProcessCrosslink(state)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(crosslinks[0], newState.CurrentCrosslinks[0]) {
		t.Errorf("Crosslink is not the same")
	}
}

func TestBaseReward_AccurateRewards(t *testing.T) {
	tests := []struct {
		a uint64
		b uint64
		c uint64
	}{
		{0, 0, 0},
		{params.BeaconConfig().MinDepositAmount, params.BeaconConfig().MinDepositAmount, 35778},
		{30 * 1e9, 30 * 1e9, 195963},
		{params.BeaconConfig().MaxDepositAmount, params.BeaconConfig().MaxDepositAmount, 202390},
		{40 * 1e9, params.BeaconConfig().MaxDepositAmount, 202390},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: tt.b}},
			Balances: []uint64{tt.a},
		}
		c := BaseReward(state, 0)
		if c != tt.c {
			t.Errorf("BaseReward(%d) = %d, want = %d",
				tt.a, c, tt.c)
		}
	}
}

func TestProcessJustificationFinalization_LessThan2ndEpoch(t *testing.T) {
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().SlotsPerEpoch,
	}
	newState, err := ProcessJustificationFinalization(state, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, newState) {
		t.Error("Did not get the original state")
	}
}

func TestProcessJustificationFinalization_CantJustifyFinalize(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxDepositAmount
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch * 2,
		PreviousJustifiedEpoch: 0,
		PreviousJustifiedRoot:  params.BeaconConfig().ZeroHash[:],
		CurrentJustifiedEpoch:  0,
		CurrentJustifiedRoot:   params.BeaconConfig().ZeroHash[:],
		ValidatorRegistry: []*pb.Validator{{ExitEpoch: e, EffectiveBalance: a}, {ExitEpoch: e, EffectiveBalance: a},
			{ExitEpoch: e, EffectiveBalance: a}, {ExitEpoch: e, EffectiveBalance: a}},
	}
	// Since Attested balances are less than total balances, nothing happened.
	newState, err := ProcessJustificationFinalization(state, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, newState) {
		t.Error("Did not get the original state")
	}
}

func TestProcessJustificationFinalization_NoBlockRootCurrentEpoch(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxDepositAmount
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch * 2,
		PreviousJustifiedEpoch: 0,
		PreviousJustifiedRoot:  params.BeaconConfig().ZeroHash[:],
		CurrentJustifiedEpoch:  0,
		CurrentJustifiedRoot:   params.BeaconConfig().ZeroHash[:],
		JustificationBitfield:  3,
		ValidatorRegistry:      []*pb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:               []uint64{a, a, a, a}, // validator total balance should be 128000000000
		LatestBlockRoots:       blockRoots,
	}
	attestedBalance := 4 * e * 3 / 2
	_, err := ProcessJustificationFinalization(state, 0, attestedBalance)
	want := "could not get block root for current epoch"
	if !strings.Contains(err.Error(), want) {
		t.Fatal("Did not receive correct error")
	}
}

func TestProcessJustificationFinalization_JustifyCurrentEpoch(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxDepositAmount
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch*2 + 1,
		PreviousJustifiedEpoch: 0,
		PreviousJustifiedRoot:  params.BeaconConfig().ZeroHash[:],
		CurrentJustifiedEpoch:  0,
		CurrentJustifiedRoot:   params.BeaconConfig().ZeroHash[:],
		JustificationBitfield:  3,
		ValidatorRegistry:      []*pb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:               []uint64{a, a, a, a}, // validator total balance should be 128000000000
		LatestBlockRoots:       blockRoots,
	}
	attestedBalance := 4 * e * 3 / 2
	newState, err := ProcessJustificationFinalization(state, 0, attestedBalance)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newState.CurrentJustifiedRoot, []byte{byte(128)}) {
		t.Errorf("Wanted current justified root: %v, got: %v",
			[]byte{byte(128)}, newState.CurrentJustifiedRoot)
	}
	if newState.CurrentJustifiedEpoch != 2 {
		t.Errorf("Wanted justified epoch: %d, got: %d",
			2, newState.CurrentJustifiedEpoch)
	}
	if !bytes.Equal(newState.FinalizedRoot, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf("Wanted current finalized root: %v, got: %v",
			params.BeaconConfig().ZeroHash, newState.FinalizedRoot)
	}
	if newState.FinalizedEpoch != 0 {
		t.Errorf("Wanted finalized epoch: %d, got: %d", 0, newState.FinalizedEpoch)
	}
}

func TestProcessJustificationFinalization_JustifyPrevEpoch(t *testing.T) {
	e := params.BeaconConfig().FarFutureEpoch
	a := params.BeaconConfig().MaxDepositAmount
	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerEpoch*2+1)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = []byte{byte(i)}
	}
	state := &pb.BeaconState{
		Slot:                   params.BeaconConfig().SlotsPerEpoch*2 + 1,
		PreviousJustifiedEpoch: 0,
		PreviousJustifiedRoot:  params.BeaconConfig().ZeroHash[:],
		CurrentJustifiedEpoch:  0,
		CurrentJustifiedRoot:   params.BeaconConfig().ZeroHash[:],
		JustificationBitfield:  3,
		ValidatorRegistry:      []*pb.Validator{{ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}, {ExitEpoch: e}},
		Balances:               []uint64{a, a, a, a}, // validator total balance should be 128000000000
		LatestBlockRoots:       blockRoots,
	}
	attestedBalance := 4 * e * 3 / 2
	newState, err := ProcessJustificationFinalization(state, attestedBalance, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(newState.CurrentJustifiedRoot, []byte{byte(128)}) {
		t.Errorf("Wanted current justified root: %v, got: %v",
			[]byte{byte(128)}, newState.CurrentJustifiedRoot)
	}
	if newState.CurrentJustifiedEpoch != 2 {
		t.Errorf("Wanted justified epoch: %d, got: %d",
			2, newState.CurrentJustifiedEpoch)
	}
	if !bytes.Equal(newState.FinalizedRoot, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf("Wanted current finalized root: %v, got: %v",
			params.BeaconConfig().ZeroHash, newState.FinalizedRoot)
	}
	if newState.FinalizedEpoch != 0 {
		t.Errorf("Wanted finalized epoch: %d, got: %d", 0, newState.FinalizedEpoch)
	}
}

func TestProcessSlashings_NotSlashed(t *testing.T) {
	s := &pb.BeaconState{
		Slot:                  0,
		ValidatorRegistry:     []*pb.Validator{{Slashed: true}},
		Balances:              []uint64{params.BeaconConfig().MaxDepositAmount},
		LatestSlashedBalances: []uint64{0, 1e9},
	}
	newState := ProcessSlashings(s)
	wanted := params.BeaconConfig().MaxDepositAmount
	if newState.Balances[0] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, newState.Balances[0])
	}
}

func TestProcessSlashings_SlashedLess(t *testing.T) {
	s := &pb.BeaconState{
		Slot: 0,
		ValidatorRegistry: []*pb.Validator{
			{Slashed: true,
				WithdrawableEpoch: params.BeaconConfig().LatestSlashedExitLength / 2,
				EffectiveBalance:  params.BeaconConfig().MaxDepositAmount},
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxDepositAmount}},
		Balances:              []uint64{params.BeaconConfig().MaxDepositAmount, params.BeaconConfig().MaxDepositAmount},
		LatestSlashedBalances: []uint64{0, 1e9},
	}
	newState := ProcessSlashings(s)
	wanted := uint64(31 * 1e9)
	if newState.Balances[0] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, newState.Balances[0])
	}
}

func TestProcessFinalUpdates_CanProcess(t *testing.T) {
	s := buildState(params.BeaconConfig().SlotsPerHistoricalRoot-1, params.BeaconConfig().SlotsPerEpoch)
	ce := helpers.CurrentEpoch(s)
	ne := ce + 1
	s.Eth1DataVotes = []*pb.Eth1Data{}
	s.Balances[0] = 29 * 1e9
	s.LatestSlashedBalances[ce] = 100
	s.LatestRandaoMixes[ce] = []byte{'A'}
	newS, err := ProcessFinalUpdates(s)
	if err != nil {
		t.Fatal(err)
	}

	// Verify effective balance is correctly updated.
	if newS.ValidatorRegistry[0].EffectiveBalance != 29*1e9 {
		t.Errorf("effective balance incorrectly updated, got %d", s.ValidatorRegistry[0].EffectiveBalance)
	}

	// Verify start shard is correctly updated.
	if newS.LatestStartShard != 64 {
		t.Errorf("start shard incorrectly updated, got %d", 64)
	}

	// Verify latest active index root is correctly updated in the right position.
	pos := (ne + params.BeaconConfig().ActivationExitDelay) % params.BeaconConfig().LatestActiveIndexRootsLength
	if bytes.Equal(newS.LatestActiveIndexRoots[pos], params.BeaconConfig().ZeroHash[:]) {
		t.Error("latest active index roots still zero hashes")
	}

	// Verify slashed balances correctly updated.
	if newS.LatestSlashedBalances[ce] != newS.LatestSlashedBalances[ne] {
		t.Errorf("wanted slashed balance %d, got %d",
			newS.LatestSlashedBalances[ce],
			newS.LatestSlashedBalances[ne])
	}

	// Verify randao is correctly updated in the right position.
	if bytes.Equal(newS.LatestRandaoMixes[ne], params.BeaconConfig().ZeroHash[:]) {
		t.Error("latest RANDAO still zero hashes")
	}

	// Verify historical root accumulator was appended.
	if len(newS.HistoricalRoots) != 1 {
		t.Errorf("wanted slashed balance %d, got %d", 1, len(newS.HistoricalRoots[ce]))
	}
}

func TestCrosslinkDelta_NoOneAttested(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	validatorCount := uint64(128)
	state := buildState(e+2, validatorCount)

	rewards, penalties, err := CrosslinkDelta(state)
	if err != nil {
		t.Fatal(err)
	}

	for i := uint64(0); i < validatorCount; i++ {
		// Since no one attested, all the validators should gain 0 reward
		if rewards[i] != 0 {
			t.Errorf("Wanted reward balance 0, got %d", rewards[i])
		}
		// Since no one attested, all the validators should get penalized the same
		if penalties[i] != BaseReward(state, i) {
			t.Errorf("Wanted penalty balance %d, got %d",
				BaseReward(state, i), penalties[i])
		}
	}
}

func TestCrosslinkDelta_SomeAttested(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	state := buildState(e+2, params.BeaconConfig().DepositsForChainStart/8)
	startShard := uint64(960)
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:              uint64(i),
				CrosslinkDataRoot: []byte{'A'},
				Shard:             startShard + 1,
			},
			InclusionDelay:      uint64(i + 100),
			AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0},
		}
	}
	state.PreviousEpochAttestations = atts
	state.CurrentCrosslinks[startShard] = &pb.Crosslink{
		CrosslinkDataRootHash32: []byte{'A'},
	}
	state.CurrentCrosslinks[startShard+1] = &pb.Crosslink{
		CrosslinkDataRootHash32: []byte{'A'},
	}

	rewards, penalties, err := CrosslinkDelta(state)
	if err != nil {
		t.Fatal(err)
	}

	committee, err := helpers.CrosslinkCommitteeAtEpoch(state, 0, startShard)
	if err != nil {
		t.Fatal(err)
	}
	_, winningIndices, err := WinningCrosslink(state, startShard+1, 0)
	if err != nil {
		t.Fatal(err)
	}
	committeeBalance := helpers.TotalBalance(state, committee)
	attestingBalance := helpers.TotalBalance(state, winningIndices)
	attestedIndices := []uint64{1932, 500, 1790, 1015, 1477, 1211, 69}
	for _, i := range attestedIndices {
		// Since all these validators attested, they should get the same rewards.
		want := BaseReward(state, i) * attestingBalance / committeeBalance
		if rewards[i] != want {
			t.Errorf("Wanted reward balance %d, got %d", want, rewards[i])
		}
		// Since all these validators attested, they shouldn't get penalized.
		if penalties[i] != 0 {
			t.Errorf("Wanted penalty balance %d, got %d",
				0, penalties[i])
		}
	}
}

func TestCrosslinkDelta_CantGetWinningCrosslink(t *testing.T) {
	state := buildState(0, 1)

	_, _, err := CrosslinkDelta(state)
	wanted := "could not get winning crosslink: could not get matching attestations"
	if !strings.Contains(err.Error(), wanted) {
		t.Fatalf("Got: %v, want: %v", err.Error(), wanted)
	}
}

func TestAttestationDelta_CantGetBlockRoot(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch

	state := buildState(2*e, 1)
	state.Slot = 0

	_, _, err := AttestationDelta(state)
	wanted := "could not get block root for epoch"
	if !strings.Contains(err.Error(), wanted) {
		t.Fatalf("Got: %v, want: %v", err.Error(), wanted)
	}
}

func TestAttestationDelta_CantGetAttestation(t *testing.T) {
	state := buildState(0, 1)

	_, _, err := AttestationDelta(state)
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
			Data: &pb.AttestationData{
				Slot: uint64(i),
			},
			InclusionDelay:      uint64(i + 100),
			AggregationBitfield: []byte{0xff},
		}
	}
	state.PreviousEpochAttestations = atts

	_, _, err := AttestationDelta(state)
	wanted := "could not get attestation indices"
	if !strings.Contains(err.Error(), wanted) {
		t.Fatalf("Got: %v, want: %v", err.Error(), wanted)
	}
}

func TestAttestationDelta_NoOneAttested(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := params.BeaconConfig().DepositsForChainStart / 32
	state := buildState(e+2, validatorCount)
	startShard := uint64(960)
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:              uint64(i),
				CrosslinkDataRoot: []byte{'A'},
				Shard:             startShard + 1,
			},
			InclusionDelay:      uint64(i + 100),
			AggregationBitfield: []byte{0xC0},
		}
	}

	rewards, penalties, err := AttestationDelta(state)
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
		wanted := 3 * BaseReward(state, i)
		if penalties[i] != wanted {
			t.Errorf("Wanted penalty balance %d, got %d",
				wanted, penalties[i])
		}
	}
}

func TestAttestationDelta_SomeAttested(t *testing.T) {
	e := params.BeaconConfig().SlotsPerEpoch
	validatorCount := params.BeaconConfig().DepositsForChainStart / 8
	state := buildState(e+2, validatorCount)
	startShard := uint64(960)
	atts := make([]*pb.PendingAttestation, 3)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:              uint64(i),
				CrosslinkDataRoot: []byte{'A'},
				Shard:             startShard + 1,
			},
			AggregationBitfield: []byte{0xC0, 0xC0, 0xC0, 0xC0},
			InclusionDelay:      1,
		}
	}
	state.PreviousEpochAttestations = atts
	state.CurrentCrosslinks[startShard] = &pb.Crosslink{
		CrosslinkDataRootHash32: []byte{'A'},
	}
	state.CurrentCrosslinks[startShard+1] = &pb.Crosslink{
		CrosslinkDataRootHash32: []byte{'A'},
	}

	rewards, penalties, err := AttestationDelta(state)
	if err != nil {
		t.Fatal(err)
	}

	//attestedIndices := []uint64{1932, 500, 1790, 1015, 1477, 1211, 69}
	attestedIndices := []uint64{500}

	attestedBalance, err := AttestingBalance(state, atts)
	totalBalance := totalActiveBalance(state)
	if err != nil {
		t.Fatal(err)
	}

	for _, i := range attestedIndices {
		// Base rewards for getting source right
		wanted := 3 * (BaseReward(state, i) * attestedBalance / totalBalance)
		// Base rewards for proposer and attesters working together getting attestation
		// on chain in the fatest manner
		wanted += BaseReward(state, i) * params.BeaconConfig().MinAttestationInclusionDelay
		if rewards[i] != wanted {
			t.Errorf("Wanted reward balance %d, got %d", wanted, rewards[i])
		}
		// Since all these validators attested, they shouldn't get penalized.
		if penalties[i] != 0 {
			t.Errorf("Wanted penalty balance %d, got %d",
				0, penalties[i])
		}
	}
}

func buildState(slot uint64, validatorCount uint64) *pb.BeaconState {
	validators := make([]*pb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxDepositAmount,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}
	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().LatestActiveIndexRootsLength,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = params.BeaconConfig().ZeroHash[:]
	}
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().LatestRandaoMixesLength,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}
	return &pb.BeaconState{
		Slot:                   slot,
		Balances:               validatorBalances,
		ValidatorRegistry:      validators,
		CurrentCrosslinks:      make([]*pb.Crosslink, params.BeaconConfig().ShardCount),
		LatestRandaoMixes:      make([][]byte, params.BeaconConfig().LatestRandaoMixesLength),
		LatestActiveIndexRoots: make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength),
		LatestSlashedBalances:  make([]uint64, params.BeaconConfig().LatestSlashedExitLength),
		LatestBlockRoots:       make([][]byte, params.BeaconConfig().SlotsPerEpoch*10),
	}
}
