package kv

import (
	"context"
	"flag"
	"reflect"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/urfave/cli/v2"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestStore_ProposerSlashingNilBucket(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	ps := &ethpb.ProposerSlashing{
		Header_1: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 1,
				ParentRoot:    make([]byte, 32),
				StateRoot:     make([]byte, 32),
				BodyRoot:      make([]byte, 32),
			},
			Signature: make([]byte, 96),
		},
		Header_2: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 1,
				ParentRoot:    make([]byte, 32),
				StateRoot:     make([]byte, 32),
				BodyRoot:      make([]byte, 32),
			},
			Signature: make([]byte, 96),
		},
	}
	has, _, err := db.HasProposerSlashing(ctx, ps)
	if err != nil {
		t.Fatalf("HasProposerSlashing should not return error: %v", err)
	}
	if has {
		t.Fatal("HasProposerSlashing should return false")
	}

	p, err := db.ProposalSlashingsByStatus(ctx, types.SlashingStatus(types.Active))
	if err != nil {
		t.Fatalf("Failed to get proposer slashing: %v", err)
	}
	if p == nil || len(p) != 0 {
		t.Fatalf("Get should return empty attester slashing array for a non existent key")
	}
}

func TestStore_SaveProposerSlashing(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	tests := []struct {
		ss types.SlashingStatus
		ps *ethpb.ProposerSlashing
	}{
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{
				Header_1: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 1,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
				Header_2: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 1,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
			},
		},
		{
			ss: types.Included,
			ps: &ethpb.ProposerSlashing{
				Header_1: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 2,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
				Header_2: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 2,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
			},
		},
		{
			ss: types.Reverted,
			ps: &ethpb.ProposerSlashing{
				Header_1: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 3,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
				Header_2: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 3,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
			},
		},
	}

	for _, tt := range tests {
		err := db.SaveProposerSlashing(ctx, tt.ss, tt.ps)
		if err != nil {
			t.Fatalf("Save proposer slashing failed: %v", err)
		}

		proposerSlashings, err := db.ProposalSlashingsByStatus(ctx, tt.ss)
		if err != nil {
			t.Fatalf("Failed to get proposer slashings: %v", err)
		}

		if proposerSlashings == nil || !reflect.DeepEqual(proposerSlashings[0], tt.ps) {
			diff, _ := messagediff.PrettyDiff(proposerSlashings[0], tt.ps)
			t.Log(diff)
			t.Fatalf("Proposer slashing: %v should be part of proposer slashings response: %v", tt.ps, proposerSlashings)
		}
	}

}

func TestStore_UpdateProposerSlashingStatus(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	tests := []struct {
		ss types.SlashingStatus
		ps *ethpb.ProposerSlashing
	}{
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{
				Header_1: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 1,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
				Header_2: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 1,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
			},
		},
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{
				Header_1: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 2,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
				Header_2: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 2,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
			},
		},
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{
				Header_1: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 3,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
				Header_2: &ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 3,
						ParentRoot:    make([]byte, 32),
						StateRoot:     make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
			},
		},
	}

	for _, tt := range tests {
		err := db.SaveProposerSlashing(ctx, tt.ss, tt.ps)
		if err != nil {
			t.Fatalf("Save proposer slashing failed: %v", err)
		}
	}

	for _, tt := range tests {
		has, st, err := db.HasProposerSlashing(ctx, tt.ps)
		if err != nil {
			t.Fatalf("Failed to get proposer slashing: %v", err)
		}
		if !has {
			t.Fatalf("Failed to find proposer slashing: %v", tt.ps)
		}
		if st != tt.ss {
			t.Fatalf("Failed to find proposer slashing with the correct status: %v", tt.ps)
		}

		err = db.SaveProposerSlashing(ctx, types.SlashingStatus(types.Included), tt.ps)
		has, st, err = db.HasProposerSlashing(ctx, tt.ps)
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
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	ps := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					ParentRoot:    make([]byte, 32),
					StateRoot:     make([]byte, 32),
					BodyRoot:      make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					ParentRoot:    make([]byte, 32),
					StateRoot:     make([]byte, 32),
					BodyRoot:      make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
		},
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 2,
					ParentRoot:    make([]byte, 32),
					StateRoot:     make([]byte, 32),
					BodyRoot:      make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 2,
					ParentRoot:    make([]byte, 32),
					StateRoot:     make([]byte, 32),
					BodyRoot:      make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
		},
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 3,
					ParentRoot:    make([]byte, 32),
					StateRoot:     make([]byte, 32),
					BodyRoot:      make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 3,
					ParentRoot:    make([]byte, 32),
					StateRoot:     make([]byte, 32),
					BodyRoot:      make([]byte, 32),
				},
				Signature: make([]byte, 96),
			},
		},
	}
	err := db.SaveProposerSlashings(ctx, types.Active, ps)
	if err != nil {
		t.Fatalf("Save proposer slashings failed: %v", err)
	}
	proposerSlashings, err := db.ProposalSlashingsByStatus(ctx, types.Active)
	if err != nil {
		t.Fatalf("Failed to get proposer slashings: %v", err)
	}
	sort.SliceStable(proposerSlashings, func(i, j int) bool {
		return proposerSlashings[i].Header_1.Header.ProposerIndex < proposerSlashings[j].Header_1.Header.ProposerIndex
	})
	if proposerSlashings == nil || !reflect.DeepEqual(proposerSlashings, ps) {
		diff, _ := messagediff.PrettyDiff(proposerSlashings, ps)
		t.Log(diff)
		t.Fatalf("Proposer slashing: %v should be part of proposer slashings response: %v", ps, proposerSlashings)
	}
}
