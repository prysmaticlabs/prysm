package epoch

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
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
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
				Target: &ethpb.Checkpoint{Epoch: 0}},
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
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
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
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
				Slot:   uint64(i),
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
		Slot:        2,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),

		Validators: validators,
		Balances:   balances,
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
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{}}},                                                       // source
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},                                        // source, target
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{Root: []byte{3}}}},                                        // source
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},                                        // source, target
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{}}},                // source, head
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{4}, Target: &ethpb.Checkpoint{}}},                           // source
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{1}}}}, // source, target, head
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{5}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},            // source, target
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{6}}}}, // source, head
	}

	currentAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{}}},                                            // none
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{2}, Target: &ethpb.Checkpoint{Root: []byte{1}}}}, // none
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
	}

	mAtts, err := MatchAttestations(state, 0)
	if err != nil {
		t.Fatal(err)
	}

	wantedSrcAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{Root: []byte{3}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{4}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{5}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{6}}}},
	}
	if !reflect.DeepEqual(mAtts.source, wantedSrcAtts) {
		t.Error("source attestations don't match")
	}

	wantedTgtAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{5}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
	}
	if !reflect.DeepEqual(mAtts.Target, wantedTgtAtts) {
		t.Error("target attestations don't match")
	}

	wantedHeadAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{1}}}},
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{6}}}},
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
	// The correct vote for target is '33'
	// The correct vote for head is '34'
	prevAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{}}},                                            // none
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{2}, Target: &ethpb.Checkpoint{Root: []byte{1}}}}, // none
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{5}, Target: &ethpb.Checkpoint{Root: []byte{1}}}}, // none
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{2}, Target: &ethpb.Checkpoint{Root: []byte{6}}}}, // none
	}

	currentAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{}}},                                                        // source
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{33}}}}, // source, target, head
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{69}, Target: &ethpb.Checkpoint{Root: []byte{33}}}},           // source, target
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{68}}}}, // source, head
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
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, Target: &ethpb.Checkpoint{}}},
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{33}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{69}, Target: &ethpb.Checkpoint{Root: []byte{33}}}},
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{68}}}},
	}
	if !reflect.DeepEqual(mAtts.source, wantedSrcAtts) {
		t.Error("source attestations don't match")
	}

	wantedTgtAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{33}}}},
		{Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{69}, Target: &ethpb.Checkpoint{Root: []byte{33}}}},
	}
	if !reflect.DeepEqual(mAtts.Target, wantedTgtAtts) {
		t.Error("target attestations don't match")
	}

	wantedHeadAtts := []*pb.PendingAttestation{
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{33}}}},
		{Data: &ethpb.AttestationData{Slot: 33, Source: &ethpb.Checkpoint{}, BeaconBlockRoot: []byte{34}, Target: &ethpb.Checkpoint{Root: []byte{68}}}},
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

func TestBaseReward_AccurateRewards(t *testing.T) {
	helpers.ClearAllCaches()

	tests := []struct {
		a uint64
		b uint64
		c uint64
	}{
		{params.BeaconConfig().MinDepositAmount, params.BeaconConfig().MinDepositAmount, 505976},
		{30 * 1e9, 30 * 1e9, 2771282},
		{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance, 2862174},
		{40 * 1e9, params.BeaconConfig().MaxEffectiveBalance, 2862174},
	}
	for _, tt := range tests {
		helpers.ClearAllCaches()
		state := &pb.BeaconState{
			Validators: []*ethpb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: tt.b}},
			Balances: []uint64{tt.a},
		}
		c, err := BaseReward(state, 0)
		if err != nil {
			t.Fatal(err)
		}
		if c != tt.c {
			t.Errorf("BaseReward(%d) = %d, want = %d",
				tt.a, c, tt.c)
		}
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

func TestProcessRegistryUpdates_NoRotation(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{ExitEpoch: params.BeaconConfig().MaxSeedLookhead},
			{ExitEpoch: params.BeaconConfig().MaxSeedLookhead},
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
		if validator.ExitEpoch != params.BeaconConfig().MaxSeedLookhead {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().MaxSeedLookhead, validator.ExitEpoch)
		}
	}
}

func TestProcessRegistryUpdates_EligibleToActivate(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                5 * params.BeaconConfig().SlotsPerEpoch,
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	limit, err := helpers.ValidatorChurnLimit(0)
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
			{ExitEpoch: params.BeaconConfig().MaxSeedLookhead,
				ActivationEpoch: 5 + params.BeaconConfig().MaxSeedLookhead + 1},
			{ExitEpoch: params.BeaconConfig().MaxSeedLookhead,
				ActivationEpoch: 5 + params.BeaconConfig().MaxSeedLookhead + 1},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	newState, err := ProcessRegistryUpdates(state)
	if err != nil {
		t.Error(err)
	}
	for i, validator := range newState.Validators {
		if validator.ExitEpoch != params.BeaconConfig().MaxSeedLookhead {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().MaxSeedLookhead, validator.ExitEpoch)
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
		if validator.ExitEpoch != params.BeaconConfig().MaxSeedLookhead+1 {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().MaxSeedLookhead+1, validator.ExitEpoch)
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
		RandaoMixes:                 make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Slashings:                   make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		BlockRoots:                  make([][]byte, params.BeaconConfig().SlotsPerEpoch*10),
		FinalizedCheckpoint:         &ethpb.Checkpoint{},
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{},
		CurrentJustifiedCheckpoint:  &ethpb.Checkpoint{},
	}
}
