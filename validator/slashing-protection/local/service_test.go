package local_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
	slashingprotection "github.com/prysmaticlabs/prysm/validator/slashing-protection"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection/local"
)

var (
	_ = slashingprotection.Protector(&local.Service{})
)

func TestAttestingHistoryForPubKey_OK(t *testing.T) {
	ctx := context.Background()
	pubKey1 := [48]byte{1}
	pubKey2 := [48]byte{2}
	validatorDB := dbtest.SetupDB(t, [][48]byte{pubKey1, pubKey2})
	history := kv.NewAttestationHistoryArray(2)
	history, err := history.SetTargetData(ctx, 1, &kv.HistoryData{Source: 0, SigningRoot: []byte{1}})
	require.NoError(t, err)
	history, err = history.SetTargetData(ctx, 2, &kv.HistoryData{Source: 1, SigningRoot: []byte{2}})
	require.NoError(t, err)
	history, err = history.SetLatestEpochWritten(ctx, 2)
	require.NoError(t, err)
	history2 := kv.NewAttestationHistoryArray(3)
	history2, err = history2.SetTargetData(ctx, 3, &kv.HistoryData{Source: 2, SigningRoot: []byte{1}})
	require.NoError(t, err)
	history2, err = history2.SetLatestEpochWritten(ctx, 3)
	require.NoError(t, err)

	histories := make(map[[48]byte]kv.EncHistoryData)
	histories[pubKey1] = history
	histories[pubKey2] = history2
	require.NoError(t, validatorDB.SaveAttestationHistoryForPubKeys(ctx, histories))

	wanted1, err := validatorDB.AttestationHistoryForPubKey(ctx, pubKey1)
	require.NoError(t, err)
	wanted2, err := validatorDB.AttestationHistoryForPubKey(ctx, pubKey2)
	require.NoError(t, err)
	require.DeepEqual(t, history, wanted1, "Unexpected retrieved history")
	require.DeepEqual(t, history2, wanted2, "Unexpected retrieved history")
}
