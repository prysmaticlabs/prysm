package blocks_test

import (
	"context"
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProcessProposerSlashings_UnmatchedHeaderSlots(t *testing.T) {
	testutil.ResetCache()
	beaconState, _ := testutil.DeterministicGenesisState(t, 20)
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          params.BeaconConfig().SlotsPerEpoch + 1,
				},
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
			},
		},
	}
	require.NoError(t, beaconState.SetSlot(currentSlot))

	b := testutil.NewBeaconBlock()
	b.Block = &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "mismatched header slots"
	_, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, b)
	assert.ErrorContains(t, want, err)
}

func TestProcessProposerSlashings_SameHeaders(t *testing.T) {
	testutil.ResetCache()
	beaconState, _ := testutil.DeterministicGenesisState(t, 2)
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
			},
		},
	}

	require.NoError(t, beaconState.SetSlot(currentSlot))
	b := testutil.NewBeaconBlock()
	b.Block = &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "expected slashing headers to differ"
	_, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, b)
	assert.ErrorContains(t, want, err)
}

func TestProcessProposerSlashings_ValidatorNotSlashable(t *testing.T) {
	registry := []*ethpb.Validator{
		{
			PublicKey:         []byte("key"),
			Slashed:           true,
			ActivationEpoch:   0,
			WithdrawableEpoch: 0,
		},
	}
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 0,
					Slot:          0,
					BodyRoot:      []byte("foo"),
				},
				Signature: bytesutil.PadTo([]byte("A"), 96),
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 0,
					Slot:          0,
					BodyRoot:      []byte("bar"),
				},
				Signature: bytesutil.PadTo([]byte("B"), 96),
			},
		},
	}

	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})
	require.NoError(t, err)
	b := testutil.NewBeaconBlock()
	b.Block = &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"validator with key %#x is not slashable",
		bytesutil.ToBytes48(beaconState.Validators()[0].PublicKey),
	)
	_, err = blocks.ProcessProposerSlashings(context.Background(), beaconState, b)
	assert.ErrorContains(t, want, err)
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	proposerIdx := uint64(1)

	header1 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerIdx,
			Slot:          0,
			ParentRoot:    make([]byte, 32),
			BodyRoot:      make([]byte, 32),
			StateRoot:     bytesutil.PadTo([]byte("A"), 32),
		},
	}
	var err error
	header1.Signature, err = helpers.ComputeDomainAndSign(beaconState, 0, header1.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerIdx])
	require.NoError(t, err)

	header2 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerIdx,
			Slot:          0,
			ParentRoot:    make([]byte, 32),
			BodyRoot:      make([]byte, 32),
			StateRoot:     bytesutil.PadTo([]byte("B"), 32),
		},
	}
	header2.Signature, err = helpers.ComputeDomainAndSign(beaconState, 0, header2.Header, params.BeaconConfig().DomainBeaconProposer, privKeys[proposerIdx])
	require.NoError(t, err)

	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: header1,
			Header_2: header2,
		},
	}

	block := testutil.NewBeaconBlock()
	block.Block.Body.ProposerSlashings = slashings

	newState, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block)
	require.NoError(t, err)

	newStateVals := newState.Validators()
	if newStateVals[1].ExitEpoch != beaconState.Validators()[1].ExitEpoch {
		t.Errorf("Proposer with index 1 did not correctly exit,"+"wanted slot:%d, got:%d",
			newStateVals[1].ExitEpoch, beaconState.Validators()[1].ExitEpoch)
	}
}

func TestVerifyProposerSlashing(t *testing.T) {
	type args struct {
		beaconState *stateTrie.BeaconState
		slashing    *ethpb.ProposerSlashing
	}

	beaconState, _ := testutil.DeterministicGenesisState(t, 2)
	currentSlot := uint64(0)
	require.NoError(t, beaconState.SetSlot(currentSlot))

	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{
			name: "same header, no signatures",
			args: args{
				slashing: &ethpb.ProposerSlashing{
					Header_1: &ethpb.SignedBeaconBlockHeader{
						Header: &ethpb.BeaconBlockHeader{
							ProposerIndex: 1,
							Slot:          0,
						},
					},
					Header_2: &ethpb.SignedBeaconBlockHeader{
						Header: &ethpb.BeaconBlockHeader{
							ProposerIndex: 1,
							Slot:          0,
						},
					},
				},
				beaconState: beaconState,
			},
			wantErr: "expected slashing headers to differ",
		},
		{ // Regression test for https://github.com/sigp/beacon-fuzz/issues/74
			name: "same header, different signatures",
			args: args{
				slashing: &ethpb.ProposerSlashing{
					Header_1: &ethpb.SignedBeaconBlockHeader{
						Header: &ethpb.BeaconBlockHeader{
							ProposerIndex: 1,
							Slot:          0,
						},
						Signature: bls.RandKey().Sign([]byte("foo")).Marshal(),
					},
					Header_2: &ethpb.SignedBeaconBlockHeader{
						Header: &ethpb.BeaconBlockHeader{
							ProposerIndex: 1,
							Slot:          0,
						},
						Signature: bls.RandKey().Sign([]byte("bar")).Marshal(),
					},
				},
				beaconState: beaconState,
			},
			wantErr: "expected slashing headers to differ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.ResetCache()
			if err := blocks.VerifyProposerSlashing(tt.args.beaconState, tt.args.slashing); (err != nil || tt.wantErr != "") && err.Error() != tt.wantErr {
				t.Errorf("VerifyProposerSlashing() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
