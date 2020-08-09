package kv

import (
	"context"
	"flag"
	"reflect"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/urfave/cli/v2"
)

func TestStore_AttesterSlashingNilBucket(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	as := &ethpb.AttesterSlashing{
		Attestation_1: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
			},
			Signature: bytesutil.PadTo([]byte("hello"), 96),
		},
		Attestation_2: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
			},
			Signature: bytesutil.PadTo([]byte("hello"), 96),
		},
	}
	has, _, err := db.HasAttesterSlashing(ctx, as)
	if err != nil {
		t.Fatalf("HasAttesterSlashing should not return error: %v", err)
	}
	if has {
		t.Fatal("HasAttesterSlashing should return false")
	}

	p, err := db.AttesterSlashings(ctx, types.SlashingStatus(types.Active))
	if err != nil {
		t.Fatalf("Failed to get attester slashing: %v", err)
	}
	if p == nil || len(p) != 0 {
		t.Fatalf("Get should return empty attester slashing array for a non existent key")
	}
}

func TestStore_SaveAttesterSlashing(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	data := &ethpb.AttestationData{
		Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		BeaconBlockRoot: make([]byte, 32),
	}
	att := &ethpb.IndexedAttestation{Data: data, Signature: make([]byte, 96)}
	tests := []struct {
		ss types.SlashingStatus
		as *ethpb.AttesterSlashing
	}{
		{
			ss: types.Active,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello"), 96)}, Attestation_2: att},
		},
		{
			ss: types.Included,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello2"), 96)}, Attestation_2: att},
		},
		{
			ss: types.Reverted,
			as: &ethpb.AttesterSlashing{Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello3"), 96)}, Attestation_2: att},
		},
	}

	for _, tt := range tests {
		err := db.SaveAttesterSlashing(ctx, tt.ss, tt.as)
		if err != nil {
			t.Fatalf("save attester slashing failed: %v", err)
		}

		attesterSlashings, err := db.AttesterSlashings(ctx, tt.ss)
		if err != nil {
			t.Fatalf("failed to get attester slashings: %v", err)
		}

		if attesterSlashings == nil || !reflect.DeepEqual(attesterSlashings[0], tt.as) {
			t.Fatalf("attester slashing: %v should be part of attester slashings response: %v", tt.as, attesterSlashings)
		}
	}

}

func TestStore_SaveAttesterSlashings(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	ckpt := &ethpb.Checkpoint{Root: make([]byte, 32)}
	data := &ethpb.AttestationData{Source: ckpt, Target: ckpt, BeaconBlockRoot: make([]byte, 32)}
	att := &ethpb.IndexedAttestation{Data: data, Signature: make([]byte, 96)}
	as := []*ethpb.AttesterSlashing{
		{Attestation_1: &ethpb.IndexedAttestation{Signature: bytesutil.PadTo([]byte("1"), 96), Data: data}, Attestation_2: att},
		{Attestation_1: &ethpb.IndexedAttestation{Signature: bytesutil.PadTo([]byte("2"), 96), Data: data}, Attestation_2: att},
		{Attestation_1: &ethpb.IndexedAttestation{Signature: bytesutil.PadTo([]byte("3"), 96), Data: data}, Attestation_2: att},
	}
	err := db.SaveAttesterSlashings(ctx, types.Active, as)
	if err != nil {
		t.Fatalf("save attester slashing failed: %v", err)
	}
	attesterSlashings, err := db.AttesterSlashings(ctx, types.Active)
	if err != nil {
		t.Fatalf("failed to get attester slashings: %v", err)
	}
	sort.SliceStable(attesterSlashings, func(i, j int) bool {
		return attesterSlashings[i].Attestation_1.Signature[0] < attesterSlashings[j].Attestation_1.Signature[0]
	})
	if attesterSlashings == nil || !reflect.DeepEqual(attesterSlashings, as) {
		t.Fatalf("Attester slashing: %v should be part of attester slashings response: %v", as, attesterSlashings)
	}
}

func TestStore_UpdateAttesterSlashingStatus(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	data := &ethpb.AttestationData{
		BeaconBlockRoot: make([]byte, 32),
		Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
	}

	tests := []struct {
		ss types.SlashingStatus
		as *ethpb.AttesterSlashing
	}{
		{
			ss: types.Active,
			as: &ethpb.AttesterSlashing{
				Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello"), 96)},
				Attestation_2: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello"), 96)},
			},
		},
		{
			ss: types.Active,
			as: &ethpb.AttesterSlashing{
				Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello2"), 96)},
				Attestation_2: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello2"), 96)},
			},
		},
		{
			ss: types.Active,
			as: &ethpb.AttesterSlashing{
				Attestation_1: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello3"), 96)},
				Attestation_2: &ethpb.IndexedAttestation{Data: data, Signature: bytesutil.PadTo([]byte("hello2"), 96)},
			},
		},
	}

	for _, tt := range tests {
		err := db.SaveAttesterSlashing(ctx, tt.ss, tt.as)
		if err != nil {
			t.Fatalf("save attester slashing failed: %v", err)
		}
	}

	for _, tt := range tests {
		has, st, err := db.HasAttesterSlashing(ctx, tt.as)
		if err != nil {
			t.Fatalf("Failed to get attester slashing: %v", err)
		}
		if !has {
			t.Fatalf("Failed to find attester slashing: %v", tt.as)
		}
		if st != tt.ss {
			t.Fatalf("Failed to find attester slashing with the correct status: %v", tt.as)
		}

		err = db.SaveAttesterSlashing(ctx, types.SlashingStatus(types.Included), tt.as)
		has, st, err = db.HasAttesterSlashing(ctx, tt.as)
		if err != nil {
			t.Fatalf("Failed to get attester slashing: %v", err)
		}
		if !has {
			t.Fatalf("Failed to find attester slashing: %v", tt.as)
		}
		if st != types.Included {
			t.Fatalf("Failed to find attester slashing with the correct status: %v", tt.as)
		}
	}
}

func TestStore_LatestEpochDetected(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	e, err := db.GetLatestEpochDetected(ctx)
	if err != nil {
		t.Fatalf("Get latest epoch detected failed: %v", err)
	}
	if e != 0 {
		t.Fatalf("Latest epoch detected should have been 0 before setting got: %d", e)
	}
	epoch := uint64(1)
	err = db.SetLatestEpochDetected(ctx, epoch)
	if err != nil {
		t.Fatalf("Set latest epoch detected failed: %v", err)
	}
	e, err = db.GetLatestEpochDetected(ctx)
	if err != nil {
		t.Fatalf("Get latest epoch detected failed: %v", err)
	}
	if e != epoch {
		t.Fatalf("Latest epoch detected should have been: %d got: %d", epoch, e)
	}
}
