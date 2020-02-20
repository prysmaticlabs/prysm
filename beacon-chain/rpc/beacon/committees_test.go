package beacon

import (
	"context"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestServer_ListBeaconCommittees_CurrentEpoch(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	helpers.ClearCache()

	numValidators := 128
	headState := setupActiveValidators(t, db, numValidators)

	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		randaoMixes[i] = make([]byte, 32)
	}
	if err := headState.SetRandaoMixes(randaoMixes); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	activeIndices, err := helpers.ActiveValidatorIndices(headState, 0)
	if err != nil {
		t.Fatal(err)
	}
	attesterSeed, err := helpers.Seed(headState, 0, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		t.Fatal(err)
	}
	committees, err := computeCommittees(0, activeIndices, attesterSeed)
	if err != nil {
		t.Fatal(err)
	}

	wanted := &ethpb.BeaconCommittees{
		Epoch:                0,
		Committees:           committees,
		ActiveValidatorCount: uint64(numValidators),
	}
	res, err := bs.ListBeaconCommittees(context.Background(), &ethpb.ListCommitteesRequest{
		QueryFilter: &ethpb.ListCommitteesRequest_Genesis{Genesis: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(res, wanted) {
		t.Errorf("Expected %v, received %v", wanted, res)
	}
}

func TestServer_ListBeaconCommittees_PreviousEpoch(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	helpers.ClearCache()

	numValidators := 128
	headState := setupActiveValidators(t, db, numValidators)

	mixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(mixes); i++ {
		mixes[i] = make([]byte, 32)
	}
	headState.SetRandaoMixes(mixes)
	headState.SetSlot(params.BeaconConfig().SlotsPerEpoch * 2)

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	activeIndices, err := helpers.ActiveValidatorIndices(headState, 1)
	if err != nil {
		t.Fatal(err)
	}
	attesterSeed, err := helpers.Seed(headState, 1, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		t.Fatal(err)
	}
	startSlot := helpers.StartSlot(1)
	wanted, err := computeCommittees(startSlot, activeIndices, attesterSeed)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		req *ethpb.ListCommitteesRequest
		res *ethpb.BeaconCommittees
	}{
		{
			req: &ethpb.ListCommitteesRequest{
				QueryFilter: &ethpb.ListCommitteesRequest_Epoch{Epoch: 1},
			},
			res: &ethpb.BeaconCommittees{
				Epoch:                1,
				Committees:           wanted,
				ActiveValidatorCount: uint64(numValidators),
			},
		},
	}
	helpers.ClearCache()
	for i, test := range tests {
		res, err := bs.ListBeaconCommittees(context.Background(), test.req)
		if err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(res, test.res) {
			diff, _ := messagediff.PrettyDiff(res, test.res)
			t.Errorf("%d/ Diff between responses %s", i, diff)
		}
	}
}

func TestServer_ListBeaconCommittees_FromArchive(t *testing.T) {
	db := dbTest.SetupDB(t)
	defer dbTest.TeardownDB(t, db)
	helpers.ClearCache()
	ctx := context.Background()

	numValidators := 128
	balances := make([]uint64, numValidators)
	validators := make([]*ethpb.Validator, 0, numValidators)
	for i := 0; i < numValidators; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		if err := db.SaveValidatorIndex(ctx, pubKey, uint64(i)); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{
			PublicKey:             pubKey,
			ActivationEpoch:       0,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			WithdrawalCredentials: make([]byte, 32),
		})
	}
	headState, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{Validators: validators, Balances: balances})
	if err != nil {
		t.Fatal(err)
	}

	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		randaoMixes[i] = make([]byte, 32)
	}
	if err := headState.SetRandaoMixes(randaoMixes); err != nil {
		t.Fatal(err)
	}

	if err := headState.SetSlot(params.BeaconConfig().SlotsPerEpoch * 10); err != nil {
		t.Fatal(err)
	}

	// Store the genesis seed.
	seed, err := helpers.Seed(headState, 0, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveArchivedCommitteeInfo(ctx, 0, &pbp2p.ArchivedCommitteeInfo{
		AttesterSeed: seed[:],
	}); err != nil {
		t.Fatal(err)
	}

	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	activeIndices, err := helpers.ActiveValidatorIndices(headState, 0)
	if err != nil {
		t.Fatal(err)
	}

	wanted, err := computeCommittees(0, activeIndices, seed)
	if err != nil {
		t.Fatal(err)
	}
	res1, err := bs.ListBeaconCommittees(context.Background(), &ethpb.ListCommitteesRequest{
		QueryFilter: &ethpb.ListCommitteesRequest_Genesis{
			Genesis: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	res2, err := bs.ListBeaconCommittees(context.Background(), &ethpb.ListCommitteesRequest{
		QueryFilter: &ethpb.ListCommitteesRequest_Epoch{
			Epoch: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res1, res2) {
		t.Fatal(err)
	}
	wantedRes := &ethpb.BeaconCommittees{
		Epoch:                0,
		Committees:           wanted,
		ActiveValidatorCount: uint64(numValidators),
	}
	if !reflect.DeepEqual(wantedRes, res1) {
		t.Errorf("Wanted %v", wantedRes)
		t.Errorf("Received %v", res1)
	}
}

func setupActiveValidators(t *testing.T, db db.Database, count int) *stateTrie.BeaconState {
	ctx := context.Background()
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.LittleEndian.PutUint64(pubKey, uint64(i))
		if err := db.SaveValidatorIndex(ctx, pubKey, uint64(i)); err != nil {
			t.Fatal(err)
		}
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{
			PublicKey:             pubKey,
			ActivationEpoch:       0,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			WithdrawalCredentials: make([]byte, 32),
		})
	}
	st, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{Validators: validators, Balances: balances})
	if err != nil {
		t.Fatal(err)
	}
	return st
}
