package db

import (
	"flag"
	"reflect"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/urfave/cli"
)

func TestStore_ProposerSlashingNilBucket(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	db := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, db)
	ps := &ethpb.ProposerSlashing{ProposerIndex: 1}
	has, _, err := db.HasProposerSlashing(ps)
	if err != nil {
		t.Fatalf("HasProposerSlashing should not return error: %v", err)
	}
	if has {
		t.Fatal("HasProposerSlashing should return false")
	}

	p, err := db.ProposalSlashingsByStatus(types.SlashingStatus(types.Active))
	if err != nil {
		t.Fatalf("Failed to get proposer slashing: %v", err)
	}
	if p == nil || len(p) != 0 {
		t.Fatalf("Get should return empty attester slashing array for a non existent key")
	}
}

func TestStore_SaveProposerSlashing(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	db := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, db)
	tests := []struct {
		ss types.SlashingStatus
		ps *ethpb.ProposerSlashing
	}{
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 1},
		},
		{
			ss: types.Included,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 2},
		},
		{
			ss: types.Reverted,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 3},
		},
	}

	for _, tt := range tests {
		err := db.SaveProposerSlashing(tt.ss, tt.ps)
		if err != nil {
			t.Fatalf("Save proposer slashing failed: %v", err)
		}

		proposerSlashings, err := db.ProposalSlashingsByStatus(tt.ss)
		if err != nil {
			t.Fatalf("Failed to get proposer slashings: %v", err)
		}

		if proposerSlashings == nil || !reflect.DeepEqual(proposerSlashings[0], tt.ps) {
			t.Fatalf("Proposer slashing: %v should be part of proposer slashings response: %v", tt.ps, proposerSlashings)
		}
	}

}

func TestStore_UpdateProposerSlashingStatus(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	db := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, db)
	tests := []struct {
		ss types.SlashingStatus
		ps *ethpb.ProposerSlashing
	}{
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 1},
		},
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 2},
		},
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{ProposerIndex: 3},
		},
	}

	for _, tt := range tests {
		err := db.SaveProposerSlashing(tt.ss, tt.ps)
		if err != nil {
			t.Fatalf("Save proposer slashing failed: %v", err)
		}
	}

	for _, tt := range tests {
		has, st, err := db.HasProposerSlashing(tt.ps)
		if err != nil {
			t.Fatalf("Failed to get proposer slashing: %v", err)
		}
		if !has {
			t.Fatalf("Failed to find proposer slashing: %v", tt.ps)
		}
		if st != tt.ss {
			t.Fatalf("Failed to find proposer slashing with the correct status: %v", tt.ps)
		}

		err = db.SaveProposerSlashing(types.SlashingStatus(types.Included), tt.ps)
		has, st, err = db.HasProposerSlashing(tt.ps)
		if err != nil {
			t.Fatalf("Failed to get proposer slashing: %v", err)
		}
		if !has {
			t.Fatalf("Failed to find proposer slashing: %v", tt.ps)
		}
		if st != types.Included {
			t.Fatalf("Failed to find proposer slashing with the correct status: %v", tt.ps)
		}

	}

}

func TestStore_SaveProposerSlashings(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	db := SetupSlasherDB(t, ctx)
	defer TeardownSlasherDB(t, db)
	ps := []*ethpb.ProposerSlashing{
		{ProposerIndex: 1},
		{ProposerIndex: 2},
		{ProposerIndex: 3},
	}
	err := db.SaveProposerSlashings(types.Active, ps)
	if err != nil {
		t.Fatalf("Save proposer slashings failed: %v", err)
	}
	proposerSlashings, err := db.ProposalSlashingsByStatus(types.Active)
	if err != nil {
		t.Fatalf("Failed to get proposer slashings: %v", err)
	}
	sort.SliceStable(proposerSlashings, func(i, j int) bool {
		return proposerSlashings[i].ProposerIndex < proposerSlashings[j].ProposerIndex
	})
	if proposerSlashings == nil || !reflect.DeepEqual(proposerSlashings, ps) {
		t.Fatalf("Proposer slashing: %v should be part of proposer slashings response: %v", ps, proposerSlashings)
	}
}
