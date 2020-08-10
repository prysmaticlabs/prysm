package kv

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStore_ProposerSlashing_CRUD(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	prop := &ethpb.ProposerSlashing{
		Header_1: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 5,
				BodyRoot:      make([]byte, 32),
				ParentRoot:    make([]byte, 32),
				StateRoot:     make([]byte, 32),
			},
			Signature: make([]byte, 96),
		},
		Header_2: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 5,
				BodyRoot:      make([]byte, 32),
				ParentRoot:    make([]byte, 32),
				StateRoot:     make([]byte, 32),
			},
			Signature: make([]byte, 96),
		},
	}
	slashingRoot, err := ssz.HashTreeRoot(prop)
	require.NoError(t, err)
	retrieved, err := db.ProposerSlashing(ctx, slashingRoot)
	require.NoError(t, err)
	assert.Equal(t, (*ethpb.ProposerSlashing)(nil), retrieved, "Expected nil proposer slashing")
	require.NoError(t, db.SaveProposerSlashing(ctx, prop))
	assert.Equal(t, true, db.HasProposerSlashing(ctx, slashingRoot), "Expected proposer slashing to exist in the db")
	retrieved, err = db.ProposerSlashing(ctx, slashingRoot)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(prop, retrieved), "Wanted %v, received %v", prop, retrieved)
	require.NoError(t, db.deleteProposerSlashing(ctx, slashingRoot))
	assert.Equal(t, false, db.HasProposerSlashing(ctx, slashingRoot), "Expected proposer slashing to have been deleted from the db")
}

func TestStore_AttesterSlashing_CRUD(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	att := &ethpb.AttesterSlashing{
		Attestation_1: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Slot:            5,
				Source: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
				Target: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
			},
			Signature: make([]byte, 96),
		},
		Attestation_2: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				BeaconBlockRoot: make([]byte, 32),
				Slot:            7,
				Source: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
				Target: &ethpb.Checkpoint{
					Epoch: 0,
					Root:  make([]byte, 32),
				},
			},
			Signature: make([]byte, 96),
		},
	}
	slashingRoot, err := ssz.HashTreeRoot(att)
	require.NoError(t, err)
	retrieved, err := db.AttesterSlashing(ctx, slashingRoot)
	require.NoError(t, err)
	assert.Equal(t, (*ethpb.AttesterSlashing)(nil), retrieved, "Expected nil attester slashing")
	require.NoError(t, db.SaveAttesterSlashing(ctx, att))
	assert.Equal(t, true, db.HasAttesterSlashing(ctx, slashingRoot), "Expected attester slashing to exist in the db")
	retrieved, err = db.AttesterSlashing(ctx, slashingRoot)
	require.NoError(t, err)
	assert.Equal(t, true, proto.Equal(att, retrieved), "Wanted %v, received %v", att, retrieved)
	require.NoError(t, db.deleteAttesterSlashing(ctx, slashingRoot))
	assert.Equal(t, false, db.HasAttesterSlashing(ctx, slashingRoot), "Expected attester slashing to have been deleted from the db")
}
