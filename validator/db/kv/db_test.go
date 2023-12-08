package kv

import (
	"context"
	"testing"

	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

type ProposalsWithPubKey struct {
	pubkey    [fieldparams.BLSPubkeyLength]byte
	proposals []Proposal
}

type AttestationWithSigningRoot struct {
	signingRoot [fieldparams.RootLength]byte
	attestation *ethpb.IndexedAttestation
}

type AttestationsWithPubKey struct {
	pubkey       [fieldparams.BLSPubkeyLength]byte
	attestations []AttestationWithSigningRoot
}

func TestStore_IsSlashingProtectionMinimal(t *testing.T) {
	testCases := []struct {
		name                    string
		shoudBeMinimal          bool
		proposalsWithPubkeys    []ProposalsWithPubKey
		attestationsWithPubkeys []AttestationsWithPubKey
	}{
		{
			name:                    "empty database",
			shoudBeMinimal:          true,
			proposalsWithPubkeys:    []ProposalsWithPubKey{},
			attestationsWithPubkeys: []AttestationsWithPubKey{},
		},
		{
			name:           "one key with multiple proposals",
			shoudBeMinimal: false,
			proposalsWithPubkeys: []ProposalsWithPubKey{
				{
					pubkey: [fieldparams.BLSPubkeyLength]byte{1},
					proposals: []Proposal{
						{
							Slot:        1,
							SigningRoot: []byte{1},
						},
						{
							Slot:        2,
							SigningRoot: []byte{2},
						},
					},
				},
			},
			attestationsWithPubkeys: []AttestationsWithPubKey{},
		},
		{
			name:                 "one key with multiple signing roots for attestations",
			shoudBeMinimal:       false,
			proposalsWithPubkeys: []ProposalsWithPubKey{},
			attestationsWithPubkeys: []AttestationsWithPubKey{
				{
					pubkey: [fieldparams.BLSPubkeyLength]byte{1},
					attestations: []AttestationWithSigningRoot{
						{
							signingRoot: [fieldparams.RootLength]byte{1},
							attestation: &ethpb.IndexedAttestation{
								Data: &ethpb.AttestationData{
									Slot: 1,
									Source: &ethpb.Checkpoint{
										Epoch: 1,
										Root:  []byte{1},
									},
									Target: &ethpb.Checkpoint{
										Epoch: 2,
										Root:  []byte{2},
									},
								},
							},
						},
						{
							signingRoot: [fieldparams.RootLength]byte{1},
							attestation: &ethpb.IndexedAttestation{
								Data: &ethpb.AttestationData{
									Slot: 100,
									Source: &ethpb.Checkpoint{
										Epoch: 2,
										Root:  []byte{1},
									},
									Target: &ethpb.Checkpoint{
										Epoch: 3,
										Root:  []byte{2},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:           "minimal database",
			shoudBeMinimal: true,
			proposalsWithPubkeys: []ProposalsWithPubKey{
				{
					pubkey: [fieldparams.BLSPubkeyLength]byte{1},
					proposals: []Proposal{
						{
							Slot:        1,
							SigningRoot: []byte{1},
						},
					},
				},
				{
					pubkey: [fieldparams.BLSPubkeyLength]byte{2},
					proposals: []Proposal{
						{
							Slot:        1,
							SigningRoot: []byte{1},
						},
					},
				},
			},
			attestationsWithPubkeys: []AttestationsWithPubKey{
				{
					pubkey: [fieldparams.BLSPubkeyLength]byte{1},
					attestations: []AttestationWithSigningRoot{
						{
							signingRoot: [fieldparams.RootLength]byte{1},
							attestation: &ethpb.IndexedAttestation{
								Data: &ethpb.AttestationData{
									Slot: 1,
									Source: &ethpb.Checkpoint{
										Epoch: 1,
										Root:  []byte{1},
									},
									Target: &ethpb.Checkpoint{
										Epoch: 2,
										Root:  []byte{2},
									},
								},
							},
						},
					},
				},
				{
					pubkey: [fieldparams.BLSPubkeyLength]byte{2},
					attestations: []AttestationWithSigningRoot{
						{
							signingRoot: [fieldparams.RootLength]byte{1},
							attestation: &ethpb.IndexedAttestation{
								Data: &ethpb.AttestationData{
									Slot: 1,
									Source: &ethpb.Checkpoint{
										Epoch: 1,
										Root:  []byte{1},
									},
									Target: &ethpb.Checkpoint{
										Epoch: 2,
										Root:  []byte{2},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Setup database
			s := setupDB(t, [][fieldparams.BLSPubkeyLength]byte{}, Complete)

			// Save proposals
			for _, ps := range tt.proposalsWithPubkeys {
				pk := ps.pubkey
				for _, p := range ps.proposals {
					err := s.SaveProposalHistoryForSlot(context.Background(), pk, p.Slot, p.SigningRoot)
					require.NoError(t, err, "Failed to save proposal history for slot")
				}
			}

			// Save attestations
			for _, at := range tt.attestationsWithPubkeys {
				pk := at.pubkey
				for _, a := range at.attestations {
					err := s.SaveAttestationForPubKey(context.Background(), pk, a.signingRoot, a.attestation)
					require.NoError(t, err, "Failed to save attestation for pubkey")
				}
			}

			// Check if the database is minimal
			isMinimal, err := IsSlashingProtectionMinimal(s)
			require.NoError(t, err, "Failed to check if database is minimal")

			require.Equal(t, tt.shoudBeMinimal, isMinimal)
		})
	}
}
