package v1_test

import (
	"encoding/hex"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v1"
)

func TestInteropListValidatingKeysZero(t *testing.T) {
	_, _, err := keymanager.NewInterop("")
	assert.ErrorContains(t, "unexpected end of JSON input", err)
}

func TestInteropListValidatingKeysEmptyJSON(t *testing.T) {
	_, _, err := keymanager.NewInterop("{}")
	assert.ErrorContains(t, "input length must be greater than 0", err)
}

func TestInteropListValidatingKeysSingle(t *testing.T) {
	direct, _, err := keymanager.NewInterop(`{"keys":1}`)
	require.NoError(t, err)
	keys, err := direct.FetchValidatingKeys()
	require.NoError(t, err)
	assert.Equal(t, 1, len(keys), "Incorrect number of keys returned")

	pkBytes, err := hex.DecodeString("25295f0d1d592a90b333e26e85149708208e9f8e8bc18f6c77bd62f8ad7a6866")
	require.NoError(t, err)
	privateKey, err := bls.SecretKeyFromBytes(pkBytes)
	require.NoError(t, err)
	assert.DeepEqual(t, privateKey.PublicKey().Marshal(), keys[0][:], "Public k 0 incorrect")
}

func TestInteropListValidatingKeysOffset(t *testing.T) {
	direct, _, err := keymanager.NewInterop(`{"keys":1,"offset":9}`)
	require.NoError(t, err)
	keys, err := direct.FetchValidatingKeys()
	require.NoError(t, err)
	assert.Equal(t, 1, len(keys), "Incorrect number of keys returned")

	pkBytes, err := hex.DecodeString("2b3b88a041168a1c4cd04bdd8de7964fd35238f95442dc678514f9dadb81ec34")
	require.NoError(t, err)
	privateKey, err := bls.SecretKeyFromBytes(pkBytes)
	require.NoError(t, err)
	require.DeepEqual(t, privateKey.PublicKey().Marshal(), keys[0][:], "Public k 0 incorrect")
}
