package bls

import (
	"math/big"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls/common"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDisallowZeroSecretKeys(t *testing.T) {
	t.Run("blst", func(t *testing.T) {
		// Blst does a zero check on the key during deserialization.
		_, err := SecretKeyFromBytes(common.ZeroSecretKey[:])
		require.Equal(t, common.ErrSecretUnmarshal, err)
	})
}

func TestDisallowZeroPublicKeys(t *testing.T) {
	t.Run("blst", func(t *testing.T) {
		_, err := PublicKeyFromBytes(common.InfinitePublicKey[:])
		require.Equal(t, common.ErrInfinitePubKey, err)
	})
}

func TestDisallowZeroPublicKeys_AggregatePubkeys(t *testing.T) {
	t.Run("blst", func(t *testing.T) {
		_, err := AggregatePublicKeys([][]byte{common.InfinitePublicKey[:], common.InfinitePublicKey[:]})
		require.Equal(t, common.ErrInfinitePubKey, err)
	})
}

func TestValidateSecretKeyString(t *testing.T) {
	t.Run("blst", func(t *testing.T) {
		zeroNum := new(big.Int).SetUint64(0)
		_, err := SecretKeyFromBigNum(zeroNum.String())
		assert.ErrorContains(t, "provided big number string sets to a key unequal to 32 bytes", err)

		rGen := rand.NewDeterministicGenerator()

		randBytes := make([]byte, 40)
		n, err := rGen.Read(randBytes)
		assert.NoError(t, err)
		assert.Equal(t, n, len(randBytes))
		rBigNum := new(big.Int).SetBytes(randBytes)

		// Expect larger than expected key size to fail.
		_, err = SecretKeyFromBigNum(rBigNum.String())
		assert.ErrorContains(t, "provided big number string sets to a key unequal to 32 bytes", err)

		key, err := RandKey()
		assert.NoError(t, err)
		rBigNum = new(big.Int).SetBytes(key.Marshal())

		// Expect correct size to pass.
		_, err = SecretKeyFromBigNum(rBigNum.String())
		assert.NoError(t, err)
	})
}
