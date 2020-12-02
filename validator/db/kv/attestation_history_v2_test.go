package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestAttestationHistoryForPubKeys_EmptyVals(t *testing.T) {
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := setupDB(t, pubkeys)
	historyForPubKeys, err := db.AttestationHistoryForPubKeysV2(context.Background(), pubkeys)
	require.NoError(t, err)
	cleanAttHistoryForPubKeys := make(map[[48]byte]EncHistoryData)
	clean := NewAttestationHistoryArray(0)
	for _, pubKey := range pubkeys {
		cleanAttHistoryForPubKeys[pubKey] = clean
	}
	require.DeepEqual(t, cleanAttHistoryForPubKeys, historyForPubKeys, "Expected attestation history epoch bits to be empty")
}

func TestAttestationHistoryForPubKeys_OK(t *testing.T) {
	ctx := context.Background()
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := setupDB(t, pubkeys)

	_, err := db.AttestationHistoryForPubKeysV2(context.Background(), pubkeys)
	require.NoError(t, err)

	setAttHistoryForPubKeys := make(map[[48]byte]EncHistoryData)
	clean := NewAttestationHistoryArray(0)
	for i, pubKey := range pubkeys {
		enc, err := clean.SetTargetData(ctx,
			10,
			&HistoryData{
				Source:      uint64(i),
				SigningRoot: []byte{1, 2, 3},
			})
		require.NoError(t, err)
		setAttHistoryForPubKeys[pubKey] = enc
	}
	err = db.SaveAttestationHistoryForPubKeysV2(context.Background(), setAttHistoryForPubKeys)
	require.NoError(t, err)
	historyForPubKeys, err := db.AttestationHistoryForPubKeysV2(context.Background(), pubkeys)
	require.NoError(t, err)
	require.DeepEqual(t, setAttHistoryForPubKeys, historyForPubKeys, "Expected attestation history epoch bits to be empty")
}

func TestAttestationHistoryForPubKey_OK(t *testing.T) {
	ctx := context.Background()
	pubkeys := [][48]byte{{30}}
	db := setupDB(t, pubkeys)

	_, err := db.AttestationHistoryForPubKeysV2(context.Background(), pubkeys)
	require.NoError(t, err)

	history := NewAttestationHistoryArray(53999)

	history, err = history.SetTargetData(ctx,
		10,
		&HistoryData{
			Source:      uint64(1),
			SigningRoot: []byte{1, 2, 3},
		})
	require.NoError(t, err)

	err = db.SaveAttestationHistoryForPubKeyV2(context.Background(), pubkeys[0], history)
	require.NoError(t, err)
	historyForPubKeys, err := db.AttestationHistoryForPubKeysV2(context.Background(), pubkeys)
	require.NoError(t, err)
	require.DeepEqual(t, history, historyForPubKeys[pubkeys[0]], "Expected attestation history epoch bits to be empty")
}
func TestStore_AttestedPublicKeys(t *testing.T) {
	ctx := context.Background()
	validatorDB, err := NewKVStore(t.TempDir(), nil)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, validatorDB.Close(), "Failed to close database")
		require.NoError(t, validatorDB.ClearDB(), "Failed to clear database")
	})

	keys, err := validatorDB.AttestedPublicKeys(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, make([][48]byte, 0), keys)

	pubKey := [48]byte{1}
	err = validatorDB.SaveAttestationHistoryForPubKeyV2(ctx, pubKey, NewAttestationHistoryArray(0))
	require.NoError(t, err)

	keys, err = validatorDB.AttestedPublicKeys(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, [][48]byte{pubKey}, keys)
}

func TestLowestSignedSourceEpoch_SaveRetrieve(t *testing.T) {
	ctx := context.Background()
	validatorDB, err := NewKVStore(t.TempDir(), nil)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, validatorDB.Close(), "Failed to close database")
		require.NoError(t, validatorDB.ClearDB(), "Failed to clear database")
	})
	p0 := [48]byte{0}
	p1 := [48]byte{1}
	// Can save.
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p0, 100))
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p1, 200))
	got, err := validatorDB.LowestSignedSourceEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(100), got)
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(200), got)

	// Can replace.
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p0, 99))
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p1, 199))
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(99), got)
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(199), got)

	// Can not replace.
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p0, 100))
	require.NoError(t, validatorDB.SaveLowestSignedSourceEpoch(ctx, p1, 200))
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(99), got)
	got, err = validatorDB.LowestSignedSourceEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(199), got)
}

func TestLowestSignedTargetEpoch_SaveRetrieveReplace(t *testing.T) {
	ctx := context.Background()
	validatorDB, err := NewKVStore(t.TempDir(), nil)
	require.NoError(t, err, "Failed to instantiate DB")
	t.Cleanup(func() {
		require.NoError(t, validatorDB.Close(), "Failed to close database")
		require.NoError(t, validatorDB.ClearDB(), "Failed to clear database")
	})
	p0 := [48]byte{0}
	p1 := [48]byte{1}
	// Can save.
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p0, 100))
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p1, 200))
	got, err := validatorDB.LowestSignedTargetEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(100), got)
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(200), got)

	// Can replace.
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p0, 99))
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p1, 199))
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(99), got)
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(199), got)

	// Can not replace.
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p0, 100))
	require.NoError(t, validatorDB.SaveLowestSignedTargetEpoch(ctx, p1, 200))
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p0)
	require.NoError(t, err)
	require.Equal(t, uint64(99), got)
	got, err = validatorDB.LowestSignedTargetEpoch(ctx, p1)
	require.NoError(t, err)
	require.Equal(t, uint64(199), got)
}
