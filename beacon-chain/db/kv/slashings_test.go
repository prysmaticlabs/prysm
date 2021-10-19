package kv

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"google.golang.org/protobuf/proto"
)

func TestStore_ProposerSlashing_CRUD(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	prop := &ethpb.ProposerSlashing{
		Header_1: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 5,
			},
		}),
		Header_2: testutil.HydrateSignedBeaconHeader(&ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 5,
			},
		}),
	}
	slashingRoot, err := prop.HashTreeRoot()
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
		Attestation_1: testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Slot: 5,
			}}),
		Attestation_2: testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Slot: 7,
			}})}
	slashingRoot, err := att.HashTreeRoot()
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
