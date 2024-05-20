package blocks

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"google.golang.org/protobuf/proto"
)

func TestPayloadGetter(t *testing.T) {
	t.Run("pre-capella does not implement payload getter", func(t *testing.T) {
		var bellatrix proto.Message = &pb.ExecutionPayload{}
		_, ok := bellatrix.(payloadGetter)
		require.Equal(t, false, ok)
	})
	t.Run("capella implements payload getter", func(t *testing.T) {
		var capella interface{} = &pb.ExecutionPayloadCapellaWithValue{}
		_, ok := capella.(payloadGetter)
		require.Equal(t, true, ok)
	})
}
