package kv

import (
	"context"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

func TestStore_ArchivedActiveValidatorChanges(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	changes := &ethpb.ArchivedActiveSetChanges{
		Activated:         nil,
		Exited:            nil,
		Ejected:           nil,
		ProposersSlashed:  nil,
		AttestersSlashed:  nil,
		VoluntaryExits:    nil,
		ProposerSlashings: nil,
		AttesterSlashings: nil,
	}
	epoch := uint64(10)
	if err := db.SaveArchivedActiveValidatorChanges(ctx, epoch, changes); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.ArchivedActiveValidatorChanges(ctx, epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(changes, retrieved) {
		t.Errorf("Wanted %v, received %v", changes, retrieved)
	}
}

func TestStore_ArchivedCommitteeInfo(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	info := &ethpb.ArchivedCommitteeInfo{}
	epoch := uint64(10)
	if err := db.SaveArchivedCommitteeInfo(ctx, epoch, info); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.ArchivedCommitteeInfo(ctx, epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(info, retrieved) {
		t.Errorf("Wanted %v, received %v", info, retrieved)
	}
}

func TestStore_ArchivedBalances(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	balances := []uint64{2, 3, 4, 5, 6, 7}
	epoch := uint64(10)
	if err := db.SaveArchivedBalances(ctx, epoch, balances); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.ArchivedBalances(ctx, epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(balances, retrieved) {
		t.Errorf("Wanted %v, received %v", balances, retrieved)
	}
}

func TestStore_ArchivedActiveIndices(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	indices := []uint64{2, 3, 4, 5, 6, 7}
	epoch := uint64(10)
	if err := db.SaveArchivedActiveIndices(ctx, epoch, indices); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.ArchivedActiveIndices(ctx, epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indices, retrieved) {
		t.Errorf("Wanted %v, received %v", indices, retrieved)
	}
}

func TestStore_ArchivedValidatorParticipation(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()
	part := &ethpb.ValidatorParticipation{}
	epoch := uint64(10)
	if err := db.SaveArchivedValidatorParticipation(ctx, epoch, part); err != nil {
		t.Fatal(err)
	}
	retrieved, err := db.ArchivedValidatorParticipation(ctx, epoch)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(part, retrieved) {
		t.Errorf("Wanted %v, received %v", part, retrieved)
	}
}
