package kv

import (
	"context"
	"testing"

	testpb "github.com/prysmaticlabs/prysm/proto/testing"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func Test_encode_handlesNilFromFunction(t *testing.T) {
	foo := func() *testpb.Puzzle {
		return nil
	}
	_, err := encode(context.Background(), foo())
	require.ErrorContains(t, "cannot encode nil message", err)
}
