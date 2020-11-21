package kv

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	attHist "github.com/prysmaticlabs/prysm/validator/slashing-protection/local/attesting-history"
)

func TestAttestationHistoryForPubKeys_EmptyVals(t *testing.T) {
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := setupDB(t, pubkeys)
	historyForPubKeys, err := db.AttestationHistoryForPubKeys(context.Background(), pubkeys)
	require.NoError(t, err)
	cleanAttHistoryForPubKeys := make(map[[48]byte]attHist.History)
	clean := attHist.New(0)
	for _, pubKey := range pubkeys {
		cleanAttHistoryForPubKeys[pubKey] = clean
	}
	require.DeepEqual(
		t,
		cleanAttHistoryForPubKeys,
		historyForPubKeys,
		"Expected attestation history epoch bits to be empty",
	)
}

func TestAttestationHistoryForPubKeys_OK(t *testing.T) {
	pubkeys := [][48]byte{{30}, {25}, {20}}
	db := setupDB(t, pubkeys)

	_, err := db.AttestationHistoryForPubKeys(context.Background(), pubkeys)
	require.NoError(t, err)

	setAttHistoryForPubKeys := make(map[[48]byte]attHist.History)
	clean := attHist.New(0)
	for i, pubKey := range pubkeys {
		enc, err := attHist.MarkAsAttested(clean, &attHist.HistoricalAttestation{
			Source:      uint64(i),
			Target:      10,
			SigningRoot: []byte{1, 2, 3},
		})
		require.NoError(t, err)
		setAttHistoryForPubKeys[pubKey] = enc
	}
	err = db.SaveAttestationHistoryForPubKeys(context.Background(), setAttHistoryForPubKeys)
	require.NoError(t, err)
	historyForPubKeys, err := db.AttestationHistoryForPubKeys(context.Background(), pubkeys)
	require.NoError(t, err)
	require.DeepEqual(t, setAttHistoryForPubKeys, historyForPubKeys, "Expected attestation history epoch bits to be empty")
}

func TestAttestationHistoryForPubKey_OK(t *testing.T) {
	pubkeys := [][48]byte{{30}}
	db := setupDB(t, pubkeys)

	_, err := db.AttestationHistoryForPubKeys(context.Background(), pubkeys)
	require.NoError(t, err)

	history := attHist.New(53999)

	newHist, err := attHist.MarkAsAttested(
		history,
		&attHist.HistoricalAttestation{
			Target:      10,
			Source:      uint64(1),
			SigningRoot: []byte{1, 2, 3},
		},
	)
	require.NoError(t, err)

	err = db.SaveAttestationHistoryForPubKey(context.Background(), pubkeys[0], newHist)
	require.NoError(t, err)
	historyForPubKeys, err := db.AttestationHistoryForPubKeys(context.Background(), pubkeys)
	require.NoError(t, err)
	require.DeepEqual(t, history, historyForPubKeys[pubkeys[0]], "Expected attestation history epoch bits to be empty")
}
