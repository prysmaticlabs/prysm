package cache

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util/random"
)

func TestSignedExecutionPayloadHeader(t *testing.T) {
	require.IsNil(t, SignedExecutionPayloadHeader())

	h := random.SignedExecutionPayloadHeader(t)
	SaveSignedExecutionPayloadHeader(h)
	require.DeepEqual(t, h, SignedExecutionPayloadHeader())
}
