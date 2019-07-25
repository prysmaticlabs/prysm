package rpc

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBeaconChainServer_ListValidatorBalances(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 100
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators, Balances: balances}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	tests := []struct {
		req *ethpb.GetValidatorBalancesRequest
		res *ethpb.ValidatorBalances
	}{
		{req: &ethpb.GetValidatorBalancesRequest{PublicKeys: [][]byte{{99}}},
			res: &ethpb.ValidatorBalances{Balances: []*ethpb.ValidatorBalances_Balance{{
				Index: 99, PublicKey: []byte{99}, Balance: 99}},
			}},
		{req: &ethpb.GetValidatorBalancesRequest{Indices: []uint64{1, 2, 3}},
			res: &ethpb.ValidatorBalances{Balances: []*ethpb.ValidatorBalances_Balance{
				{Index: 1, PublicKey: []byte{1}, Balance: 1},
				{Index: 2, PublicKey: []byte{2}, Balance: 2},
				{Index: 3, PublicKey: []byte{3}, Balance: 3}},
			}},
		{req: &ethpb.GetValidatorBalancesRequest{PublicKeys: [][]byte{{10}, {11}, {12}}},
			res: &ethpb.ValidatorBalances{Balances: []*ethpb.ValidatorBalances_Balance{
				{Index: 10, PublicKey: []byte{10}, Balance: 10},
				{Index: 11, PublicKey: []byte{11}, Balance: 11},
				{Index: 12, PublicKey: []byte{12}, Balance: 12}},
			}},
		{req: &ethpb.GetValidatorBalancesRequest{PublicKeys: [][]byte{{2}, {3}}, Indices: []uint64{3, 4}}, // Duplication
			res: &ethpb.ValidatorBalances{Balances: []*ethpb.ValidatorBalances_Balance{
				{Index: 2, PublicKey: []byte{2}, Balance: 2},
				{Index: 3, PublicKey: []byte{3}, Balance: 3},
				{Index: 4, PublicKey: []byte{4}, Balance: 4}},
			}},
	}

	for _, test := range tests {
		res, err := bs.ListValidatorBalances(context.Background(), test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Error("Incorrect respond of validator balances")
		}
	}
}

func TestBeaconChainServer_ListValidatorBalancesOutOfRange(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 1
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators, Balances: balances}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.GetValidatorBalancesRequest{Indices: []uint64{uint64(count)}}
	wanted := fmt.Sprintf("validator index %d >= balance list %d", count, len(balances))
	if _, err := bs.ListValidatorBalances(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_GetValidatorsNoPagination(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 100
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	received, err := bs.GetValidators(context.Background(), &ethpb.GetValidatorsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(validators, received.Validators) {
		fmt.Println(received.Validators)
		t.Fatal("Incorrect respond of validators")
	}
}

func TestBeaconChainServer_GetValidatorsPagination(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 100
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators, Balances: balances}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	tests := []struct {
		req *ethpb.GetValidatorsRequest
		res *ethpb.Validators
	}{
		{req: &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(1), PageSize: 3},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{3}},
					{PublicKey: []byte{4}},
					{PublicKey: []byte{5}}},
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(count)}},
		{req: &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(10), PageSize: 5},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{50}},
					{PublicKey: []byte{51}},
					{PublicKey: []byte{52}},
					{PublicKey: []byte{53}},
					{PublicKey: []byte{54}}},
				NextPageToken: strconv.Itoa(11),
				TotalSize:     int32(count)}},
		{req: &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(33), PageSize: 3},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{99}}},
				NextPageToken: strconv.Itoa(34),
				TotalSize:     int32(count)}},
		{req: &ethpb.GetValidatorsRequest{PageSize: 2},
			res: &ethpb.Validators{
				Validators: []*ethpb.Validator{
					{PublicKey: []byte{0}},
					{PublicKey: []byte{1}}},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     int32(count)}},
	}
	for _, test := range tests {
		res, err := bs.GetValidators(context.Background(), test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			t.Error("Incorrect respond of validators")
		}
	}
}

func TestBeaconChainServer_GetValidatorsPaginationOutOfRange(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 1
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(1), PageSize: 100}
	wanted := fmt.Sprintf("page start %d >= validator list %d", req.PageSize, len(validators))
	if _, err := bs.GetValidators(context.Background(), req); !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestBeaconChainServer_GetValidatorsMaxPageSize(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 1000
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	start := 1
	max := params.BeaconConfig().MaxPageSize
	exceedsMax := int32(max + 1)
	req := &ethpb.GetValidatorsRequest{PageToken: strconv.Itoa(start), PageSize: exceedsMax}
	res, err := bs.GetValidators(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	i := start * max
	j := (start + 1) * max
	if !reflect.DeepEqual(res.Validators, validators[i:j]) {
		t.Error("Incorrect respond of validators")
	}
}

func TestBeaconChainServer_GetValidatorsDefaultPageSize(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	count := 1000
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		if err := db.SaveValidatorIndex([]byte{byte(i)}, i); err != nil {
			t.Fatal(err)
		}
		validators = append(validators, &ethpb.Validator{PublicKey: []byte{byte(i)}})
	}

	if err := db.SaveState(
		context.Background(),
		&pbp2p.BeaconState{Validators: validators}); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	req := &ethpb.GetValidatorsRequest{}
	res, err := bs.GetValidators(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	j := params.BeaconConfig().DefaultPageSize
	if !reflect.DeepEqual(res.Validators, validators[i:j]) {
		t.Error("Incorrect respond of validators")
	}
}

func TestBeaconChainServer_GetValidatorsParticipation(t *testing.T) {
	helpers.ClearAllCaches()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	epoch := uint64(1)
	attestedBalance := uint64(1)
	validatorCount := uint64(100)

	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	atts := []*pbp2p.PendingAttestation{{Data: &ethpb.AttestationData{Crosslink: &ethpb.Crosslink{Shard: 0}, Target: &ethpb.Checkpoint{}}}}
	var crosslinks []*ethpb.Crosslink
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &ethpb.Crosslink{
			StartEpoch: 0,
			DataRoot:   []byte{'A'},
		})
	}

	s := &pbp2p.BeaconState{
		Slot:                       epoch*params.BeaconConfig().SlotsPerEpoch + 1,
		Validators:                 validators,
		Balances:                   balances,
		BlockRoots:                 make([][]byte, 128),
		Slashings:                  []uint64{0, 1e9, 1e9},
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:           make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CompactCommitteesRoots:     make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentCrosslinks:          crosslinks,
		CurrentEpochAttestations:   atts,
		FinalizedCheckpoint:        &ethpb.Checkpoint{},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{},
	}

	bs := &BeaconChainServer{
		beaconDB: db,
	}

	if err := bs.beaconDB.SaveState(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	if err := bs.beaconDB.SaveFinalizedBlock(&ethpb.BeaconBlock{Slot: 1}); err != nil {
		t.Fatal(err)
	}

	res, err := bs.GetValidatorParticipation(context.Background(), &ethpb.GetValidatorParticipationRequest{Epoch: epoch})
	if err != nil {
		t.Fatal(err)
	}

	wanted := &ethpb.ValidatorParticipation{
		Epoch:                   epoch,
		VotedEther:              attestedBalance,
		EligibleEther:           validatorCount * params.BeaconConfig().MaxEffectiveBalance,
		GlobalParticipationRate: float32(attestedBalance) / float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance),
	}

	if !reflect.DeepEqual(res, wanted) {
		t.Error("Incorrect validator participation respond")
	}
}
