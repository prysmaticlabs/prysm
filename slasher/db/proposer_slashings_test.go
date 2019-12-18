package db

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func TestStore_ProposerSlashingNilBucket(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)
	ps := &ethpb.ProposerSlashing{ProposerIndex: 1}
	has, _, err := db.HasProposerSlashing(ps)
	if err != nil {
		t.Fatalf("HasProposerSlashing should not return error: %v", err)
	}
	if has {
		t.Fatal("HasProposerSlashing should return false")
	}

	p, err := db.ProposerSlashings(SlashingStatus(Active))
	if err != nil {
		t.Fatalf("failed to get proposer slashing: %v", err)
	}
	if p != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestStore_SaveProposerSlashing(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)
	tests := []struct {
		ss SlashingStatus
		ps *ethpb.ProposerSlashing
	}{
		{
			ss: Active,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 1},
		},
		{
			ss: Included,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 2},
		},
		{
			ss: Reverted,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 3},
		},
	}

	for _, tt := range tests {
		err := db.SaveProposerSlashing(tt.ss, tt.ps)
		if err != nil {
			t.Fatalf("save proposer slashing failed: %v", err)
		}

		proposerSlashings, err := db.ProposerSlashings(tt.ss)
		if err != nil {
			t.Fatalf("failed to get proposer slashings: %v", err)
		}

		if proposerSlashings == nil || !reflect.DeepEqual(proposerSlashings[0], tt.ps) {
			t.Fatalf("proposer slashing: %v should be part of proposer slashings response: %v", tt.ps, proposerSlashings)
		}
	}

}

func TestStore_DeleteProposerSlashingWithStatus(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)
	tests := []struct {
		ss SlashingStatus
		ps *ethpb.ProposerSlashing
	}{
		{
			ss: Active,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 1},
		},
		{
			ss: Included,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 2},
		},
		{
			ss: Reverted,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 3},
		},
	}

	for _, tt := range tests {
		err := db.SaveProposerSlashing(tt.ss, tt.ps)
		if err != nil {
			t.Fatalf("save proposer slashing failed: %v", err)
		}
	}

	for _, tt := range tests {
		has, _, err := db.HasProposerSlashing(tt.ps)
		if err != nil {
			t.Fatalf("failed to get proposer slashing: %v", err)
		}
		if !has {
			t.Fatalf("failed to find proposer slashing: %v", tt.ps)
		}

		err = db.DeleteProposerSlashingWithStatus(tt.ss, tt.ps)
		if err != nil {
			t.Fatalf("delete proposer slashings failed: %v", err)
		}
		has, _, err = db.HasProposerSlashing(tt.ps)
		if err != nil {
			t.Fatalf("error while trying to get non existing proposer slashing: %v", err)
		}
		if has {
			t.Fatalf("proposer slashing: %v should have been deleted", tt.ps)
		}

	}

}

func TestStore_UpdateProposerSlashingStatus(t *testing.T) {
	db := SetupSlasherDB(t)
	defer TeardownSlasherDB(t, db)
	tests := []struct {
		ss SlashingStatus
		ps *ethpb.ProposerSlashing
	}{
		{
			ss: Active,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 1},
		},
		{
			ss: Active,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 2},
		},
		{
			ss: Active,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 3},
		},
	}

	for _, tt := range tests {
		err := db.SaveProposerSlashing(tt.ss, tt.ps)
		if err != nil {
			t.Fatalf("save proposer slashing failed: %v", err)
		}
	}

	for _, tt := range tests {
		has, st, err := db.HasProposerSlashing(tt.ps)
		if err != nil {
			t.Fatalf("failed to get proposer slashing: %v", err)
		}
		if !has {
			t.Fatalf("failed to find proposer slashing: %v", tt.ps)
		}
		if st != tt.ss {
			t.Fatalf("failed to find proposer slashing with the correct status: %v", tt.ps)
		}

		err = db.SaveProposerSlashing(SlashingStatus(Included), tt.ps)
		has, st, err = db.HasProposerSlashing(tt.ps)
		if err != nil {
			t.Fatalf("failed to get proposer slashing: %v", err)
		}
		if !has {
			t.Fatalf("failed to find proposer slashing: %v", tt.ps)
		}
		if st != Included {
			t.Fatalf("failed to find proposer slashing with the correct status: %v", tt.ps)
		}

	}

}
