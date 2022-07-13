package blockchain

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestIsInvalidBlock(t *testing.T) {
	require.Equal(t, true, IsInvalidBlock(ErrInvalidPayload)) // Already wrapped.
	err := invalidBlock{error: ErrInvalidPayload}
	require.Equal(t, true, IsInvalidBlock(err))

	newErr := errors.Wrap(err, "wrap me")
	require.Equal(t, true, IsInvalidBlock(newErr))
}

func TestInvalidBlockRoot(t *testing.T) {
	require.Equal(t, [32]byte{}, InvalidBlockRoot(ErrUndefinedExecutionEngineError))
	require.Equal(t, [32]byte{}, InvalidBlockRoot(ErrInvalidPayload))

	err := invalidBlock{error: ErrInvalidPayload, root: [32]byte{'a'}}
	require.Equal(t, [32]byte{'a'}, InvalidBlockRoot(err))
}
