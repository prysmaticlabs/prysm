package forkchoice

import (
	"context"
	"encoding/binary"
	"reflect"
	"testing"

	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestDeleteValidatorIdx_DeleteWorks(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	epoch := uint64(2)
	v.InsertActivatedIndices(epoch+1, []uint64{0, 1, 2})
	v.InsertExitedVal(epoch+1, []uint64{0, 2})
	var validators []*ethpb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		Validators: validators,
		Slot:       epoch * params.BeaconConfig().SlotsPerEpoch,
	}

	if err := saveValidatorIdx(ctx, state, db); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}
	if err := deleteValidatorIdx(ctx, state, db); err != nil {
		t.Fatalf("Could not delete validator idx: %v", err)
	}
	wantedIdx := uint64(1)
	idx, _, err := db.ValidatorIndex(ctx, bytesutil.ToBytes48(validators[wantedIdx].PublicKey))
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	wantedIdx = uint64(2)
	if db.HasValidatorIndex(ctx, bytesutil.ToBytes48(validators[wantedIdx].PublicKey)) {
		t.Errorf("Validator index %d should have been deleted", wantedIdx)
	}
	if v.ExitedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}

func TestSaveValidatorIdx_SaveRetrieveWorks(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	epoch := uint64(1)
	v.InsertActivatedIndices(epoch+1, []uint64{0, 1, 2})
	var validators []*ethpb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		Validators: validators,
		Slot:       epoch * params.BeaconConfig().SlotsPerEpoch,
	}

	if err := saveValidatorIdx(ctx, state, db); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}

	wantedIdx := uint64(2)
	idx, _, err := db.ValidatorIndex(ctx, bytesutil.ToBytes48(validators[wantedIdx].PublicKey))
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	if v.ActivatedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}
}

func TestSaveValidatorIdx_IdxNotInState(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	kv := db.(*kv.Store)
	defer testDB.TeardownDB(t, kv)

	epoch := uint64(100)
	// Tried to insert 5 active indices to DB with only 3 validators in state
	v.InsertActivatedIndices(epoch+1, []uint64{0, 1, 2, 3, 4})
	var validators []*ethpb.Validator
	for i := 0; i < 3; i++ {
		pubKeyBuf := make([]byte, params.BeaconConfig().BLSPubkeyLength)
		binary.PutUvarint(pubKeyBuf, uint64(i))
		validators = append(validators, &ethpb.Validator{
			PublicKey: pubKeyBuf,
		})
	}
	state := &pb.BeaconState{
		Validators: validators,
		Slot:       epoch * params.BeaconConfig().SlotsPerEpoch,
	}

	if err := saveValidatorIdx(ctx, state, db); err != nil {
		t.Fatalf("Could not save validator idx: %v", err)
	}

	wantedIdx := uint64(2)
	idx, _, err := db.ValidatorIndex(ctx, bytesutil.ToBytes48(validators[wantedIdx].PublicKey))
	if err != nil {
		t.Fatalf("Could not get validator index: %v", err)
	}
	if wantedIdx != idx {
		t.Errorf("Wanted: %d, got: %d", wantedIdx, idx)
	}

	if v.ActivatedValFromEpoch(epoch) != nil {
		t.Errorf("Activated validators mapping for epoch %d still there", epoch)
	}

	// Verify the skipped validators are included in the next epoch
	if !reflect.DeepEqual(v.ActivatedValFromEpoch(epoch+2), []uint64{3, 4}) {
		t.Error("Did not get wanted validator from activation queue")
	}
}
