package kv

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

type publicKeyTestStruct struct {
	validatorIndex types.ValidatorIndex
	pk             []byte
}

var pkTests []publicKeyTestStruct

func init() {
	pkTests = []publicKeyTestStruct{
		{
			validatorIndex: 1,
			pk:             []byte{1, 2, 3},
		},
		{
			validatorIndex: 2,
			pk:             []byte{4, 5, 6},
		},
		{
			validatorIndex: 3,
			pk:             []byte{7, 8, 9},
		},
	}
}

func TestNilDBValidatorPublicKey(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	validatorIndex := types.ValidatorIndex(1)

	pk, err := db.ValidatorPubKey(ctx, validatorIndex)
	require.NoError(t, err, "Nil ValidatorPubKey should not return error")
	require.DeepEqual(t, []uint8(nil), pk)
}

func TestSavePubKey(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	for _, tt := range pkTests {
		err := db.SavePubKey(ctx, tt.validatorIndex, tt.pk)
		require.NoError(t, err, "Save validator public key failed")

		pk, err := db.ValidatorPubKey(ctx, tt.validatorIndex)
		require.NoError(t, err, "Failed to get validator public key")
		require.NotNil(t, pk)
		require.DeepEqual(t, tt.pk, pk, "Should return validator public key")
	}
}

func TestDeletePublicKey(t *testing.T) {

	db := setupDB(t)
	ctx := context.Background()

	for _, tt := range pkTests {
		require.NoError(t, db.SavePubKey(ctx, tt.validatorIndex, tt.pk), "Save validator public key failed")
	}

	for _, tt := range pkTests {
		pk, err := db.ValidatorPubKey(ctx, tt.validatorIndex)
		require.NoError(t, err, "Failed to get validator public key")
		require.NotNil(t, pk)
		require.DeepEqual(t, tt.pk, pk, "Should return validator public key")

		err = db.DeletePubKey(ctx, tt.validatorIndex)
		require.NoError(t, err, "Delete validator public key")
		pk, err = db.ValidatorPubKey(ctx, tt.validatorIndex)
		require.NoError(t, err)
		require.DeepEqual(t, []byte(nil), pk, "Expected validator public key to be deleted")
	}
}
