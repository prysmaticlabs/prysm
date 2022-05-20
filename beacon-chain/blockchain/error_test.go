package blockchain

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestIsInvalidBlock(t *testing.T) {
	require.Equal(t, false, IsInvalidBlock(ErrInvalidPayload))
	err := invalidBlock{ErrInvalidPayload}
	require.Equal(t, true, IsInvalidBlock(err))

	newErr := errors.Wrap(err, "wrap me")
	require.Equal(t, true, IsInvalidBlock(newErr))
}
