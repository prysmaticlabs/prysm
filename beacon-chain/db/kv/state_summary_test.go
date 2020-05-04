package kv

import (
	"context"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestStateSummary_CanSaveRretrieve(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	r1 := bytesutil.ToBytes32([]byte{'A'})
	r2 := bytesutil.ToBytes32([]byte{'B'})
	s1 := &pb.StateSummary{Slot: 1, Root: r1[:]}

	// State summary should not exist yet.
	if db.HasStateSummary(ctx, r1) {
		t.Fatal("State summary should not be saved")
	}

	if err := db.SaveStateSummary(ctx, s1); err != nil {
		t.Fatal(err)
	}
	if !db.HasStateSummary(ctx, r1) {
		t.Fatal("State summary should be saved")
	}

	saved, err := db.StateSummary(ctx, r1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, s1) {
		t.Error("State summary does not equal")
	}

	// Save a new state summary.
	s2 := &pb.StateSummary{Slot: 2, Root: r2[:]}

	// State summary should not exist yet.
	if db.HasStateSummary(ctx, r2) {
		t.Fatal("State summary should not be saved")
	}

	if err := db.SaveStateSummary(ctx, s2); err != nil {
		t.Fatal(err)
	}
	if !db.HasStateSummary(ctx, r2) {
		t.Fatal("State summary should be saved")
	}

	saved, err = db.StateSummary(ctx, r2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, s2) {
		t.Error("State summary does not equal")
	}
}
