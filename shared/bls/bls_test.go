package bls

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDisallowZeroSecretKeys(t *testing.T) {
	zeroKey := [32]byte{}

	_, err := SecretKeyFromBytes(zeroKey[:])
	require.NotNil(t, err)
}
