package v1_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v1"
)

func TestDirectListValidatingKeysNil(t *testing.T) {
	direct := keymanager.NewDirect(nil)
	keys, err := direct.FetchValidatingKeys()
	require.NoError(t, err)
	assert.Equal(t, 0, len(keys), "Incorrect number of keys returned")
}

func TestDirectListValidatingKeysSingle(t *testing.T) {
	sks := make([]bls.SecretKey, 0)
	sks = append(sks, bls.RandKey())
	direct := keymanager.NewDirect(sks)
	keys, err := direct.FetchValidatingKeys()
	require.NoError(t, err)
	assert.Equal(t, 1, len(keys), "Incorrect number of keys returned")
}

func TestDirectListValidatingKeysMultiple(t *testing.T) {
	sks := make([]bls.SecretKey, 0)
	numKeys := 256
	for i := 0; i < numKeys; i++ {
		sks = append(sks, bls.RandKey())
	}
	direct := keymanager.NewDirect(sks)
	keys, err := direct.FetchValidatingKeys()
	require.NoError(t, err)
	assert.Equal(t, numKeys, len(keys), "Incorrect number of keys returned")
}

func TestSignNoSuchKey(t *testing.T) {
	sks := make([]bls.SecretKey, 0)
	direct := keymanager.NewDirect(sks)
	_, err := direct.Sign([48]byte{}, [32]byte{})
	assert.ErrorContains(t, keymanager.ErrNoSuchKey.Error(), err)
}

func TestSign(t *testing.T) {
	sks := make([]bls.SecretKey, 0)
	sks = append(sks, bls.RandKey())
	direct := keymanager.NewDirect(sks)

	pubKey := bytesutil.ToBytes48(sks[0].PublicKey().Marshal())
	msg := [32]byte{}
	sig, err := direct.Sign(pubKey, msg)
	require.NoError(t, err)
	require.Equal(t, true, sig.Verify(sks[0].PublicKey(), bytesutil.FromBytes32(msg)), "Failed to verify generated signature")
}
