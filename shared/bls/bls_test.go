package bls

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls/common"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDisallowZeroSecretKeys(t *testing.T) {
    t.Run("herumi", func(t *testing.T) {
        flags := &featureconfig.Flags{}
        reset := featureconfig.InitWithReset(flags)
        defer reset()

        _, err := SecretKeyFromBytes(common.ZeroSecretKey[:])
        require.Equal(t, err, common.ErrZeroKey)
    })


    t.Run("blst", func(t *testing.T) {
        flags.EnableBlst = true
        reset = featureconfig.InitWithReset(flags)
        defer reset()

        _, err = SecretKeyFromBytes(common.ZeroSecretKey[:])
        require.Equal(t, err, common.ErrZeroKey)
    })
}

func TestDisallowZeroPublicKeys(t *testing.T) {
    t.Run("herumi", func(t *testing.T) {
        flags := &featureconfig.Flags{}
        reset := featureconfig.InitWithReset(flags)
        defer reset()

        _, err := PublicKeyFromBytes(common.InfinitePublicKey[:])
        require.Equal(t, err, common.ErrInfinitePubKey)
    })

    t.Run("blst", func(t *testing.T) {
        flags.EnableBlst = true
        reset = featureconfig.InitWithReset(flags)
        defer reset()

        _, err = PublicKeyFromBytes(common.InfinitePublicKey[:])
        require.Equal(t, err, common.ErrInfinitePubKey)
    })
}
