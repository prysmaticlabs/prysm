package bls

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls/common"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDisallowZeroSecretKeys(t *testing.T) {
	flags := &featureconfig.Flags{}
	t.Run("herumi", func(t *testing.T) {
		flags := &featureconfig.Flags{}
		reset := featureconfig.InitWithReset(flags)
		defer reset()

		_, err := SecretKeyFromBytes(common.ZeroSecretKey[:])
		require.Equal(t, common.ErrZeroKey, err)
	})

	t.Run("blst", func(t *testing.T) {
		flags.EnableBlst = true
		reset := featureconfig.InitWithReset(flags)
		defer reset()
		// Blst does a zero check on the key during deserialization.
		_, err := SecretKeyFromBytes(common.ZeroSecretKey[:])
		require.Equal(t, common.ErrSecretUnmarshal, err)
	})
}

func TestDisallowZeroPublicKeys(t *testing.T) {
	flags := &featureconfig.Flags{}

	t.Run("herumi", func(t *testing.T) {
		reset := featureconfig.InitWithReset(flags)
		defer reset()

		_, err := PublicKeyFromBytes(common.InfinitePublicKey[:])
		require.Equal(t, common.ErrInfinitePubKey, err)
	})

	t.Run("blst", func(t *testing.T) {
		flags.EnableBlst = true
		reset := featureconfig.InitWithReset(flags)
		defer reset()

		_, err := PublicKeyFromBytes(common.InfinitePublicKey[:])
		require.Equal(t, common.ErrInfinitePubKey, err)
	})
}

func TestDisallowZeroPublicKeys_AggregatePubkeys(t *testing.T) {
	flags := &featureconfig.Flags{}

	t.Run("herumi", func(t *testing.T) {
		reset := featureconfig.InitWithReset(flags)
		defer reset()

		_, err := AggregatePublicKeys([][]byte{common.InfinitePublicKey[:], common.InfinitePublicKey[:]})
		require.Equal(t, common.ErrInfinitePubKey, err)
	})

	t.Run("blst", func(t *testing.T) {
		flags.EnableBlst = true
		reset := featureconfig.InitWithReset(flags)
		defer reset()

		_, err := AggregatePublicKeys([][]byte{common.InfinitePublicKey[:], common.InfinitePublicKey[:]})
		require.Equal(t, common.ErrInfinitePubKey, err)
	})
}
