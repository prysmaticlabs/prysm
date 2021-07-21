package kv

import (
	"context"
	"sort"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sszutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestStore_ProposerSlashingNilBucket(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	ps := &ethpb.ProposerSlashing{
		Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 1,
			},
		}),
		Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 1,
			},
		}),
	}
	has, _, err := db.HasProposerSlashing(ctx, ps)
	require.NoError(t, err)
	require.Equal(t, false, has)

	p, err := db.ProposalSlashingsByStatus(ctx, types.SlashingStatus(types.Active))
	require.NoError(t, err, "Failed to get proposer slashing")
	require.NotNil(t, p)
	require.Equal(t, 0, len(p), "Get should return empty attester slashing array for a non existent key")
}

func TestStore_SaveProposerSlashing(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		ss types.SlashingStatus
		ps *ethpb.ProposerSlashing
	}{
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{
				Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 1,
					},
				}),
				Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 1,
					},
				}),
			},
		},
		{
			ss: types.Included,
			ps: &ethpb.ProposerSlashing{
				Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 2,
					},
				}),
				Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 2,
					},
				}),
			},
		},
		{
			ss: types.Reverted,
			ps: &ethpb.ProposerSlashing{
				Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 3,
					},
				}),
				Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 3,
					},
				}),
			},
		},
	}

	for _, tt := range tests {
		err := db.SaveProposerSlashing(ctx, tt.ss, tt.ps)
		require.NoError(t, err, "Save proposer slashing failed")

		proposerSlashings, err := db.ProposalSlashingsByStatus(ctx, tt.ss)
		require.NoError(t, err, "Failed to get proposer slashings")

		var diff string
		if len(proposerSlashings) > 0 {
			diff, _ = messagediff.PrettyDiff(proposerSlashings[0], tt.ps)
		} else {
			diff, _ = messagediff.PrettyDiff(nil, tt.ps)
		}
		t.Log(diff)

		if len(proposerSlashings) == 0 || !sszutil.DeepEqual(proposerSlashings[0], tt.ps) {
			t.Fatalf("Proposer slashing: %v should be part of proposer slashings response: %v", tt.ps, proposerSlashings)
		}
	}
}

func TestStore_UpdateProposerSlashingStatus(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		ss types.SlashingStatus
		ps *ethpb.ProposerSlashing
	}{
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{
				Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 1,
					},
				}),
				Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 1,
					},
				}),
			},
		},
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{
				Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 2,
					},
				}),
				Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 2,
					},
				}),
			},
		},
		{
			ss: types.Active,
			ps: &ethpb.ProposerSlashing{
				Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 3,
					},
				}),
				Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
					Header: &ethpb.BeaconBlockHeader{
						ProposerIndex: 3,
					},
				}),
			},
		},
	}

	for _, tt := range tests {
		err := db.SaveProposerSlashing(ctx, tt.ss, tt.ps)
		require.NoError(t, err, "Save proposer slashing failed")
	}

	for _, tt := range tests {
		has, st, err := db.HasProposerSlashing(ctx, tt.ps)
		require.NoError(t, err, "Failed to get proposer slashing")
		require.Equal(t, true, has, "Failed to find proposer slashing")
		require.Equal(t, tt.ss, st, "Failed to find proposer slashing with the correct status")

		err = db.SaveProposerSlashing(ctx, types.SlashingStatus(types.Included), tt.ps)
		require.NoError(t, err)
		has, st, err = db.HasProposerSlashing(ctx, tt.ps)
		require.NoError(t, err, "Failed to get proposer slashing")
		require.Equal(t, true, has, "Failed to find proposer slashing")
		require.Equal(t, types.SlashingStatus(types.Included), st, "Failed to find proposer slashing with the correct status")
	}
}

func TestStore_SaveProposerSlashings(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	ps := []*ethpb.ProposerSlashing{
		{
			Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
				},
			}),
			Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
				},
			}),
		},
		{
			Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 2,
				},
			}),
			Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 2,
				},
			}),
		},
		{
			Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 3,
				},
			}),
			Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 3,
				},
			}),
		},
	}
	err := db.SaveProposerSlashings(ctx, types.Active, ps)
	require.NoError(t, err, "Save proposer slashings failed")
	proposerSlashings, err := db.ProposalSlashingsByStatus(ctx, types.Active)
	require.NoError(t, err, "Failed to get proposer slashings")
	sort.SliceStable(proposerSlashings, func(i, j int) bool {
		return proposerSlashings[i].Header_1.Header.ProposerIndex < proposerSlashings[j].Header_1.Header.ProposerIndex
	})
	if proposerSlashings == nil || !sszutil.DeepEqual(proposerSlashings, ps) {
		diff, _ := messagediff.PrettyDiff(proposerSlashings, ps)
		t.Log(diff)
		t.Fatalf("Proposer slashing: %v should be part of proposer slashings response: %v", ps, proposerSlashings)
	}
}
