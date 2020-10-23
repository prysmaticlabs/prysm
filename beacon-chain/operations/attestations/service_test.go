package attestations

import (
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStop_OK(t *testing.T) {
	s, serviceCtx, err := NewService(&Config{})
	require.NoError(t, err)
	require.NoError(t, s.Stop(serviceCtx.Ctx), "Unable to stop attestation pool service")
}

func TestStatus_Error(t *testing.T) {
	err := errors.New("bad bad bad")
	s := &Service{err: err}
	assert.ErrorContains(t, s.err.Error(), s.Status())
}
