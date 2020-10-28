package bls

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bls/common"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDisallowZeroSecretKeys(t *testing.T) {
	flags := &featureconfig.Flags{}
	reset := featureconfig.InitWithReset(flags)

	_, err := SecretKeyFromBytes(common.ZeroSecretKey[:])
	require.Equal(t, err, common.ErrZeroKey)
	reset()

	flags.EnableBlst = true
	reset = featureconfig.InitWithReset(flags)

	_, err = SecretKeyFromBytes(common.ZeroSecretKey[:])
	require.Equal(t, err, common.ErrZeroKey)
	reset()
}

func TestDisallowZeroPublicKeys(t *testing.T) {
	flags := &featureconfig.Flags{}
	reset := featureconfig.InitWithReset(flags)

	_, err := PublicKeyFromBytes(common.InfinitePublicKey[:])
	require.Equal(t, err, common.ErrInfinitePubKey)
	reset()

	flags.EnableBlst = true
	reset = featureconfig.InitWithReset(flags)

	_, err = PublicKeyFromBytes(common.InfinitePublicKey[:])
	require.Equal(t, err, common.ErrInfinitePubKey)
	reset()
}
