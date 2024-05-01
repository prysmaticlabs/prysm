package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util/random"
)

func TestStore_SignedExecutionPayloadEnvelopeBlind(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	_, err := db.SignedExecutionPayloadEnvelopeBlind(ctx, []byte("test"))
	require.ErrorIs(t, err, ErrNotFound)

	env := random.SignedExecutionPayloadEnvelopeBlind(t)
	err = db.SaveSignedExecutionPayloadEnvelopeBlind(ctx, env)
	require.NoError(t, err)
	got, err := db.SignedExecutionPayloadEnvelopeBlind(ctx, env.Message.BeaconBlockRoot)
	require.NoError(t, err)
	require.DeepEqual(t, got, env)
}
